package sync

import (
	"sort"
	"time"
)

type FileInfo struct {
	Key     string
	Size    int64
	ModTime time.Time
}

type ActionKind string

const (
	ActionUpload       ActionKind = "upload"
	ActionDownload     ActionKind = "download"
	ActionDeleteRemote ActionKind = "delete"
)

type Action struct {
	Kind ActionKind
	Key  string
	Size int64
}

type Plan struct {
	Actions   []Action
	Unchanged int
}

// PlanUpload decides which local files to push. A file is uploaded when it is
// missing remotely, its size differs, or it was modified after the remote copy
// was written.
func PlanUpload(local, remote map[string]FileInfo, deleteExtra bool) Plan {
	var plan Plan
	for key, lf := range local {
		rf, ok := remote[key]
		if ok && rf.Size == lf.Size && !lf.ModTime.After(rf.ModTime) {
			plan.Unchanged++
			continue
		}
		plan.Actions = append(plan.Actions, Action{Kind: ActionUpload, Key: key, Size: lf.Size})
	}
	if deleteExtra {
		for key, rf := range remote {
			if _, ok := local[key]; !ok {
				plan.Actions = append(plan.Actions, Action{Kind: ActionDeleteRemote, Key: key, Size: rf.Size})
			}
		}
	}
	sortActions(plan.Actions)
	return plan
}

// PlanDownload decides which remote objects to pull. Object mtimes reflect
// upload time rather than the original file mtime, so only presence and size
// are compared.
func PlanDownload(remote, local map[string]FileInfo) Plan {
	var plan Plan
	for key, rf := range remote {
		lf, ok := local[key]
		if ok && lf.Size == rf.Size {
			plan.Unchanged++
			continue
		}
		plan.Actions = append(plan.Actions, Action{Kind: ActionDownload, Key: key, Size: rf.Size})
	}
	sortActions(plan.Actions)
	return plan
}

func sortActions(actions []Action) {
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].Kind != actions[j].Kind {
			return actions[i].Kind < actions[j].Kind
		}
		return actions[i].Key < actions[j].Key
	})
}
