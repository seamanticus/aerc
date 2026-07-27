//go:build notmuch
// +build notmuch

package notmuch

import (
	"fmt"
	"strings"

	"git.sr.ht/~rjarry/aerc/models"
	"git.sr.ht/~rjarry/aerc/worker/types"
	"git.sr.ht/~rjarry/go-opt/v2"
)

type queryBuilder struct {
	s string
}

func (q *queryBuilder) and(s string) {
	if len(s) == 0 {
		return
	}
	if len(q.s) != 0 {
		q.s += " and "
	}
	q.s += "(" + s + ")"
}

func (q *queryBuilder) or(s string) {
	if len(s) == 0 {
		return
	}
	if len(q.s) != 0 {
		q.s += " or "
	}
	q.s += "(" + s + ")"
}

func invert(s string, invert bool) string {
	if len(s) == 0 || !invert {
		return s
	}
	return "not (" + s + ")"
}

func translate(criteria *types.SearchCriteria) string {
	if criteria == nil {
		return ""
	}
	var base queryBuilder
	translatePart(&base, criteria.Match)
	translatePart(&base, criteria.Exclude)
	return base.s
}

func translatePart(base *queryBuilder, crit *types.SearchCriteriaPart) {
	if crit == nil {
		return
	}

	// recipients
	var from queryBuilder
	for _, f := range crit.From {
		from.or("from:" + opt.QuoteArg(f))
	}
	if from.s != "" {
		base.and(invert(from.s, crit.Invert))
	}

	var to queryBuilder
	for _, t := range crit.To {
		to.or("to:" + opt.QuoteArg(t))
	}
	if to.s != "" {
		base.and(invert(to.s, crit.Invert))
	}

	var cc queryBuilder
	for _, c := range crit.Cc {
		cc.or("cc:" + opt.QuoteArg(c))
	}
	if cc.s != "" {
		base.and(invert(cc.s, crit.Invert))
	}

	// flags
	for f := range flagToTag {
		if crit.WithFlags.Has(f) {
			base.and(invert(getParsedFlag(f, false), crit.Invert))
		}
		if crit.WithoutFlags.Has(f) {
			base.and(invert(getParsedFlag(f, true), crit.Invert))
		}
	}

	// dates
	switch {
	case !crit.StartDate.IsZero() && !crit.EndDate.IsZero():
		base.and(invert(fmt.Sprintf("date:@%d..@%d",
			crit.StartDate.Unix(), crit.EndDate.Unix()), crit.Invert))
	case !crit.StartDate.IsZero():
		base.and(invert(fmt.Sprintf("date:@%d..", crit.StartDate.Unix()), crit.Invert))
	case !crit.EndDate.IsZero():
		base.and(invert(fmt.Sprintf("date:..@%d", crit.EndDate.Unix()), crit.Invert))
	}

	// other terms
	if len(crit.Terms) > 0 {
		if crit.SearchBody {
			base.and(invert("body:"+opt.QuoteArg(strings.Join(crit.Terms, " ")), crit.Invert))
		} else {
			for _, term := range crit.Terms {
				base.and(invert(term, crit.Invert))
			}
		}
	}
}

func getParsedFlag(flag models.Flags, inverse bool) string {
	name := "tag:" + flagToTag[flag]
	if flagToInvert[flag] {
		name = "not " + name
	}
	return name
}
