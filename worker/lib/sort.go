package lib

import (
	"cmp"
	"slices"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"github.com/emersion/go-message/mail"
)

func Sort(messageInfos []*models.MessageInfo,
	criteria []*types.SortCriterion,
) ([]models.UID, error) {
	// loop through in reverse to ensure we sort by non-primary fields first
	for i := len(criteria) - 1; i >= 0; i-- {
		criterion := criteria[i]
		switch criterion.Field {
		case types.SortArrival:
			sortSlice(criterion, messageInfos, func(a, b *models.MessageInfo) int {
				return a.InternalDate.Compare(b.InternalDate)
			})
		case types.SortCc:
			sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*mail.Address {
					return msgInfo.Envelope.Cc
				})
		case types.SortDate:
			sortSlice(criterion, messageInfos, func(a, b *models.MessageInfo) int {
				return a.Envelope.Date.Compare(b.Envelope.Date)
			})
		case types.SortFrom:
			sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*mail.Address {
					return msgInfo.Envelope.From
				})
		case types.SortRead:
			sortFlags(messageInfos, criterion, models.SeenFlag)
		case types.SortFlagged:
			sortFlags(messageInfos, criterion, models.FlaggedFlag)
		case types.SortSize:
			sortSlice(criterion, messageInfos, func(a, b *models.MessageInfo) int {
				return cmp.Compare(a.Size, b.Size)
			})
		case types.SortSubject:
			sortStrings(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) string {
					subject := strings.ToLower(msgInfo.Envelope.Subject)
					subject = strings.TrimPrefix(subject, "re: ")
					return strings.TrimPrefix(subject, "fwd: ")
				})
		case types.SortTo:
			sortAddresses(messageInfos, criterion,
				func(msgInfo *models.MessageInfo) []*mail.Address {
					return msgInfo.Envelope.To
				})
		}
	}
	var uids []models.UID
	// copy in reverse as msgList displays backwards
	for i := len(messageInfos) - 1; i >= 0; i-- {
		uids = append(uids, messageInfos[i].Uid)
	}
	return uids, nil
}

func sortAddresses(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) []*mail.Address,
) {
	sortSlice(criterion, messageInfos, func(a, b *models.MessageInfo) int {
		addressA, addressB := getValue(a), getValue(b)
		var firstA, firstB *mail.Address
		if len(addressA) > 0 {
			firstA = addressA[0]
		}
		if len(addressB) > 0 {
			firstB = addressB[0]
		}
		if firstA != nil && firstB != nil {
			getName := func(addr *mail.Address) string {
				if addr.Name != "" {
					return addr.Name
				}
				return addr.Address
			}
			return cmp.Compare(getName(firstA), getName(firstB))
		}
		if firstA != nil {
			return -1
		}
		if firstB != nil {
			return 1
		}
		return 0
	})
}

func sortFlags(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	testFlag models.Flags,
) {
	var slice []*boolStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &boolStore{
			Value:   msgInfo.Flags.Has(testFlag),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, slice, func(a, b *boolStore) int {
		if a.Value == b.Value {
			return 0
		}
		if a.Value {
			return -1
		}
		return 1
	})
	for i := range messageInfos {
		messageInfos[i] = slice[i].MsgInfo
	}
}

func sortStrings(messageInfos []*models.MessageInfo, criterion *types.SortCriterion,
	getValue func(*models.MessageInfo) string,
) {
	var slice []*lexiStore
	for _, msgInfo := range messageInfos {
		slice = append(slice, &lexiStore{
			Value:   getValue(msgInfo),
			MsgInfo: msgInfo,
		})
	}
	sortSlice(criterion, slice, func(a, b *lexiStore) int {
		return cmp.Compare(a.Value, b.Value)
	})
	for i := range messageInfos {
		messageInfos[i] = slice[i].MsgInfo
	}
}

type lexiStore struct {
	Value   string
	MsgInfo *models.MessageInfo
}

type boolStore struct {
	Value   bool
	MsgInfo *models.MessageInfo
}

func sortSlice[E any](criterion *types.SortCriterion, slice []E, compare func(a, b E) int) {
	if criterion.Reverse {
		slices.SortStableFunc(slice, func(a, b E) int {
			return compare(b, a)
		})
	} else {
		slices.SortStableFunc(slice, compare)
	}
}
