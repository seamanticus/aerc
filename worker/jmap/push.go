package jmap

import (
	"context"
	"fmt"
	"slices"
	"time"

	"git.sr.ht/~rjarry/aerc/lib/log"
	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/jmap/cache"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rockorager/go-jmap"
	"git.sr.ht/~rockorager/go-jmap/core/push"
	"git.sr.ht/~rockorager/go-jmap/mail/email"
	"git.sr.ht/~rockorager/go-jmap/mail/mailbox"
	"git.sr.ht/~rockorager/go-jmap/mail/thread"
)

func (w *JMAPWorker) monitorChanges() {
	defer log.PanicHandler()

	events := push.EventSource{
		Client:  w.client,
		Handler: w.handleChange,
		Ping:    uint(w.config.serverPing.Seconds()),
	}

	w.stop = make(chan struct{})
	go func() {
		defer log.PanicHandler()
		<-w.stop
		w.w.Errorf("listen stopping")
		w.stop = nil
		events.Close()
	}()

	for w.stop != nil {
		w.w.Debugf("listening for changes")
		err := events.Listen()
		if err != nil {
			w.w.PostMessage(&types.Error{
				Error: fmt.Errorf("jmap listen: %w", err),
			}, nil)
			time.Sleep(5 * time.Second)
		}
	}
}

const (
	MailboxState = "Mailbox"
	EmailState   = "Email"
	ThreadState  = "Thread"
)

func (w *JMAPWorker) handleChange(s *jmap.StateChange) {
	newState, ok := s.Changed[w.AccountId()]
	if !ok {
		return
	}
	w.w.Debugf("state change %#v", newState)

	var req jmap.Request

	mboxState, err := w.cache.GetMailboxState()
	if err != nil {
		w.w.Debugf("GetMailboxState: %s", err)
	}
	if mboxState != "" && newState[MailboxState] != mboxState {
		callID := req.Invoke(&mailbox.Changes{
			Account:    w.AccountId(),
			SinceState: mboxState,
		})
		req.Invoke(&mailbox.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Mailbox/changes",
				Path:     "/created",
			},
		})
		req.Invoke(&mailbox.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Mailbox/changes",
				Path:     "/updated",
			},
		})
	}

	queryChangesCalls := make(map[string]jmap.ID)
	folderContents := make(map[jmap.ID]*cache.FolderContents)
	for id := range w.mboxes {
		contents, err := w.cache.GetFolderContents(id)
		if err != nil {
			continue
		}
		filter, err := w.translateSearch(id, contents.Filter)
		if err != nil {
			continue
		}
		callID := req.Invoke(&email.QueryChanges{
			Account:         w.AccountId(),
			Filter:          filter,
			Sort:            translateSort(contents.Sort),
			SinceQueryState: contents.QueryState,
		})
		queryChangesCalls[callID] = id
		folderContents[id] = contents
	}

	emailState, err := w.cache.GetEmailState()
	if err != nil {
		w.w.Debugf("GetEmailState: %s", err)
	}
	if emailState != "" && newState[EmailState] != emailState {
		callID := req.Invoke(&email.Changes{
			Account:    w.AccountId(),
			SinceState: emailState,
		})
		req.Invoke(&email.Get{
			Account:        w.AccountId(),
			Properties:     emailProperties,
			BodyProperties: bodyProperties,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Email/changes",
				Path:     "/updated",
			},
		})
		req.Invoke(&email.Get{
			Account:        w.AccountId(),
			Properties:     emailProperties,
			BodyProperties: bodyProperties,
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Email/changes",
				Path:     "/created",
			},
		})
	}

	threadState, err := w.cache.GetThreadState()
	if err != nil {
		w.w.Debugf("GetThreadState: %s", err)
	}
	if threadState != "" && newState[ThreadState] != threadState {
		callID := req.Invoke(&thread.Changes{
			Account:    w.AccountId(),
			SinceState: threadState,
		})
		req.Invoke(&thread.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Thread/changes",
				Path:     "/created",
			},
		})
		req.Invoke(&thread.Get{
			Account: w.AccountId(),
			ReferenceIDs: &jmap.ResultReference{
				ResultOf: callID,
				Name:     "Thread/changes",
				Path:     "/updated",
			},
		})
	}

	if len(req.Calls) == 0 {
		return
	}

	resp, err := w.Do(context.TODO(), &req)
	if err != nil {
		w.w.Errorf("handleChange: %v", err)
		return
	}

	var labelsChanged bool
	// threadEmails are email IDs from threads which changed or were
	// created
	var threadEmails []jmap.ID
	// addedEmails are genuinely new messages per mailbox from QueryChanges,
	// used to set Index and RecentFlag in GetResponse.
	// email ID -> mailbox ID -> position index in message store
	addedEmails := make(map[jmap.ID]map[jmap.ID]int)
	// removedEmails are messages removed from mailboxes by QueryChanges,
	// used to skip re-posting in GetResponse.
	// email ID -> mailbox ID -> true
	removedEmails := make(map[jmap.ID]map[jmap.ID]bool)

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *mailbox.ChangesResponse:
			for _, id := range r.Destroyed {
				dir, ok := w.mbox2dir[id]
				if ok {
					w.w.PostMessage(&types.RemoveDirectory{
						Directory: dir,
					}, nil)
				}
				w.deleteMbox(id)
				err = w.cache.DeleteMailbox(id)
				if err != nil {
					w.w.Warnf("DeleteMailbox: %s", err)
				}
				labelsChanged = true
			}
			err = w.cache.PutMailboxState(r.NewState)
			if err != nil {
				w.w.Warnf("PutMailboxState: %s", err)
			}

		case *mailbox.GetResponse:
			for _, mbox := range r.List {
				if mbox.Role == mailbox.RoleArchive && w.config.useLabels {
					continue
				}
				err = w.cache.PutMailbox(mbox.ID, mbox)
				if err != nil {
					w.w.Warnf("PutMailbox: %s", err)
				}
				m, exist := w.mboxes[mbox.ID]
				if exist && mbox.Name != m.Name {
					w.w.PostMessage(&types.RemoveDirectory{
						Directory: w.mbox2dir[mbox.ID],
					}, nil)
					w.deleteMbox(mbox.ID)
					labelsChanged = true
					exist = false
				}
				if exist {
					w.mboxes[mbox.ID] = mbox
					w.w.PostMessage(&types.DirectoryInfo{
						Info: &models.DirectoryInfo{
							Name:   w.mbox2dir[mbox.ID],
							Exists: int(mbox.TotalEmails),
							Unseen: int(mbox.UnreadEmails),
						},
					}, nil)
				} else {
					w.addMbox(mbox)
					w.w.PostMessage(&types.Directory{
						Dir: &models.Directory{
							Name:   w.mbox2dir[mbox.ID],
							Role:   jmapRole2aerc[mbox.Role],
							Exists: int(mbox.TotalEmails),
							Unseen: int(mbox.UnreadEmails),
						},
					}, nil)
					labelsChanged = true
				}
			}
			err = w.cache.PutMailboxState(r.State)
			if err != nil {
				w.w.Warnf("PutMailboxState: %s", err)
			}

		case *email.QueryChangesResponse:
			mboxId := queryChangesCalls[inv.CallID]
			contents := folderContents[mboxId]
			dir := w.mbox2dir[mboxId]

			removed := make(map[jmap.ID]models.UID)
			for _, id := range r.Removed {
				removed[id] = models.UID(id)
			}
			for _, add := range r.Added {
				if _, ok := removed[add.ID]; ok {
					// message just changed, ignore
					delete(removed, add.ID)
				} else {
					// Store add index for each new message
					mboxes, ok := addedEmails[add.ID]
					if !ok {
						mboxes = make(map[jmap.ID]int)
						addedEmails[add.ID] = mboxes
					}
					mboxes[mboxId] = int(add.Index)
				}
			}

			// Rebuild the cached message ID list using
			// the original Removed/Added from the response
			// (not the deduped `removed` map).
			if contents != nil {
				allRemoved := make(map[jmap.ID]bool)
				for _, id := range r.Removed {
					allRemoved[id] = true
				}
				added := make(map[int]jmap.ID)
				for _, add := range r.Added {
					added[int(add.Index)] = add.ID
				}
				n := len(contents.MessageIDs) - len(allRemoved) + len(added)
				if n < 0 {
					w.w.Errorf("bug: invalid folder contents state")
					err = w.cache.DeleteFolderContents(mboxId)
					if err != nil {
						w.w.Warnf("DeleteFolderContents: %s", err)
					}
				} else {
					ids := make([]jmap.ID, 0, n)
					i := 0
					for _, id := range contents.MessageIDs {
						if allRemoved[id] {
							continue
						}
						if addedId, ok := added[i]; ok {
							ids = append(ids, addedId)
							delete(added, i)
							i += 1
						}
						ids = append(ids, id)
						i += 1
					}
					for _, id := range added {
						ids = append(ids, id)
					}
					contents.MessageIDs = ids
					contents.QueryState = r.NewQueryState
					err = w.cache.PutFolderContents(mboxId, contents)
					if err != nil {
						w.w.Warnf("PutFolderContents: %s", err)
					}
				}
			}

			// Post MessagesDeleted for removed messages
			if len(removed) > 0 {
				deletedUids := make([]models.UID, 0, len(removed))
				for id, uid := range removed {
					deletedUids = append(deletedUids, uid)
					mboxes, ok := removedEmails[id]
					if !ok {
						mboxes = make(map[jmap.ID]bool)
						removedEmails[id] = mboxes
					}
					mboxes[mboxId] = true
				}
				w.w.PostMessage(&types.MessagesDeleted{
					Directory: dir,
					Uids:      deletedUids,
				}, nil)
			}

		case *thread.ChangesResponse:
			for _, id := range r.Destroyed {
				err = w.cache.DeleteThread(id)
				if err != nil {
					w.w.Warnf("DeleteThread: %s", err)
				}
			}
			err = w.cache.PutThreadState(r.NewState)
			if err != nil {
				w.w.Warnf("PutThreadState: %s", err)
			}

		case *thread.GetResponse:
			for _, thread := range r.List {
				err = w.cache.PutThread(thread.ID, thread.EmailIDs)
				if err != nil {
					w.w.Warnf("PutThread: %s", err)
				}
				// We keep the list of all emails and check in a
				// subsequent request which ones we need to
				// fetch
				threadEmails = append(threadEmails, thread.EmailIDs...)
			}
			err = w.cache.PutThreadState(r.State)
			if err != nil {
				w.w.Warnf("PutThreadState: %s", err)
			}

		case *email.GetResponse:
			for _, m := range r.List {
				old, err := w.cache.GetEmail(m.ID)
				if err == nil {
					for mboxId := range old.MailboxIDs {
						if !m.MailboxIDs[mboxId] {
							dir, ok := w.mbox2dir[mboxId]
							if !ok {
								continue
							}
							w.w.PostMessage(&types.MessagesDeleted{
								Directory: dir,
								Uids:      []models.UID{models.UID(m.ID)},
							}, nil)
						}
					}
				}
				err = w.cache.PutEmail(m.ID, m)
				if err != nil {
					w.w.Warnf("PutEmail: %s", err)
				}
				removedBoxes := removedEmails[m.ID]
				indexes, ok := addedEmails[m.ID]
				if ok {
					delete(addedEmails, m.ID)
				}
				for mboxId := range m.MailboxIDs {
					dir, ok := w.mbox2dir[mboxId]
					if !ok {
						continue
					}
					if removedBoxes != nil && removedBoxes[mboxId] {
						continue
					}
					info := w.translateMsgInfo(m, dir)
					if indexes != nil {
						i, ok := indexes[mboxId]
						if ok {
							info.Index = &i
							// Set recent on created messages so we
							// get a notification
							if !info.Flags.Has(models.SeenFlag) {
								info.Flags |= models.RecentFlag
							}
						}
					}
					w.w.PostMessage(&types.MessageInfo{
						Info: info,
					}, nil)
				}
			}
			err = w.cache.PutEmailState(r.State)
			if err != nil {
				w.w.Warnf("PutEmailState: %s", err)
			}

		case *jmap.MethodError:
			w.w.Errorf("%s: %s", wrapMethodError(r))
			if inv.Name == "Email/queryChanges" {
				id := queryChangesCalls[inv.CallID]
				w.w.Infof("flushing %q contents from cache",
					w.mbox2dir[id])
				err := w.cache.DeleteFolderContents(id)
				if err != nil {
					w.w.Warnf("DeleteFolderContents: %s", err)
				}
			}
		}
	}

	if w.config.useLabels && labelsChanged {
		labels := make([]string, 0, len(w.dir2mbox))
		for dir := range w.dir2mbox {
			labels = append(labels, dir)
		}
		slices.Sort(labels)
		w.w.PostMessage(&types.LabelList{Labels: labels}, nil)
	}

	w.refreshQueriesAndThreads(addedEmails, threadEmails)
}

// refreshQueriesAndThreads updates the cached query for any mailbox which was updated
func (w *JMAPWorker) refreshQueriesAndThreads(
	addedEmails map[jmap.ID]map[jmap.ID]int,
	threadEmails []jmap.ID,
) {
	if len(addedEmails) == 0 && len(threadEmails) == 0 {
		return
	}

	ids := make([]jmap.ID, 0, len(addedEmails)+len(threadEmails))
	for i := range addedEmails {
		ids = append(ids, i)
	}
	ids = append(ids, threadEmails...)

	var req jmap.Request

	req.Invoke(&email.Get{
		Account:        w.AccountId(),
		Properties:     emailProperties,
		BodyProperties: bodyProperties,
		IDs:            ids,
	})

	resp, err := w.Do(context.TODO(), &req)
	if err != nil {
		w.w.Errorf("%s", err)
		return
	}

	for _, inv := range resp.Responses {
		switch r := inv.Args.(type) {
		case *email.GetResponse:
			for _, m := range r.List {
				old, err := w.cache.GetEmail(m.ID)
				if err == nil {
					for mboxId := range old.MailboxIDs {
						if !m.MailboxIDs[mboxId] {
							dir, ok := w.mbox2dir[mboxId]
							if !ok {
								continue
							}
							w.w.PostMessage(&types.MessagesDeleted{
								Directory: dir,
								Uids:      []models.UID{models.UID(m.ID)},
							}, nil)
						}
					}
				}
				err = w.cache.PutEmail(m.ID, m)
				if err != nil {
					w.w.Warnf("PutEmail: %s", err)
				}
				indexes := addedEmails[m.ID]
				for mboxId := range m.MailboxIDs {
					dir, ok := w.mbox2dir[mboxId]
					if !ok {
						continue
					}
					info := w.translateMsgInfo(m, dir)
					if indexes != nil {
						i, ok := indexes[mboxId]
						if ok {
							info.Index = &i
							// Set recent on created messages so we
							// get a notification
							if !info.Flags.Has(models.SeenFlag) {
								info.Flags |= models.RecentFlag
							}
						}
					}
					w.w.PostMessage(&types.MessageInfo{
						Info: info,
					}, nil)
				}
			}
			err = w.cache.PutEmailState(r.State)
			if err != nil {
				w.w.Warnf("PutEmailState: %s", err)
			}

		case *jmap.MethodError:
			w.w.Errorf("%s", wrapMethodError(r))
		}
	}
}
