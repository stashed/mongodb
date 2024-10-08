/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conditions

import (
	"sort"

	kmapi "kmodules.xyz/client-go/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// localizedCondition defines a condition with the information of the object the util
// was originated from.
type localizedCondition struct {
	*kmapi.Condition
	Getter
}

// merge a list of condition into a single one.
// This operation is designed to ensure visibility of the most relevant util for defining the
// operational state of a component. E.g. If there is one error in the condition list, this one takes
// priority over the other util, and it should be reflected in the target condition.
//
// More specifically:
// 1. Conditions are grouped by status, severity
// 2. The resulting condition groups are sorted according to the following priority:
//   - P0 - Status=False, Severity=Error
//   - P1 - Status=False, Severity=Warning
//   - P2 - Status=False, Severity=Info
//   - P3 - Status=True
//   - P4 - Status=Unknown
//
// 3. The group with the highest priority is used to determine status, severity and other info of the target condition.
//
// Please note that the last operation includes also the task of computing the Reason and the Message for the target
// condition; in order to complete such task some trade-off should be made, because there is no a golden rule
// for summarizing many Reason/Message into single Reason/Message.
// mergeOptions allows the user to adapt this process to the specific needs by exposing a set of merge strategies.
func merge(conditions []localizedCondition, targetCondition kmapi.ConditionType, options *mergeOptions) *kmapi.Condition {
	g := getConditionGroups(conditions)
	if len(g) == 0 {
		return nil
	}

	if g.TopGroup().status == metav1.ConditionTrue {
		return TrueCondition(targetCondition)
	}

	targetReason := getReason(g, options)
	targetMessage := getMessage(g, options)

	if g.TopGroup().status == metav1.ConditionFalse {
		return FalseCondition(targetCondition, targetReason, g.TopGroup().severity, targetMessage) //nolint:govet
	}
	return UnknownCondition(targetCondition, targetReason, targetMessage) //nolint:govet
}

// getConditionGroups groups a list of conditions according to status, severity values.
// Additionally, the resulting groups are sorted by mergePriority.
func getConditionGroups(conditions []localizedCondition) conditionGroups {
	groups := conditionGroups{}

	for _, condition := range conditions {
		if condition.Condition == nil {
			continue
		}

		added := false
		for i := range groups {
			if groups[i].status == condition.Status && groups[i].severity == condition.Severity {
				groups[i].conditions = append(groups[i].conditions, condition)
				added = true
				break
			}
		}
		if !added {
			groups = append(groups, conditionGroup{
				conditions: []localizedCondition{condition},
				status:     condition.Status,
				severity:   condition.Severity,
			})
		}
	}

	// sort groups by priority
	sort.Sort(groups)

	// sorts conditions in the TopGroup, so we ensure predictable result for merge strategies.
	// condition are sorted using the same lexicographic order used by Set; in case two conditions
	// have the same type, condition are sorted using according to the alphabetical order of the source object name.
	if len(groups) > 0 {
		sort.Slice(groups[0].conditions, func(i, j int) bool {
			a := groups[0].conditions[i]
			b := groups[0].conditions[j]
			if a.Type != b.Type {
				return lexicographicLess(a.Condition, b.Condition)
			}
			return a.GetName() < b.GetName()
		})
	}

	return groups
}

// conditionGroups provides supports for grouping a list of conditions to be
// merged into a single condition. ConditionGroups can be sorted by mergePriority.
type conditionGroups []conditionGroup

func (g conditionGroups) Len() int {
	return len(g)
}

func (g conditionGroups) Less(i, j int) bool {
	return g[i].mergePriority() < g[j].mergePriority()
}

func (g conditionGroups) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

// TopGroup returns the condition group with the highest mergePriority.
func (g conditionGroups) TopGroup() *conditionGroup {
	if len(g) == 0 {
		return nil
	}
	return &g[0]
}

// TrueGroup returns the condition group with status True, if any.
func (g conditionGroups) TrueGroup() *conditionGroup {
	return g.getByStatusAndSeverity(metav1.ConditionTrue, kmapi.ConditionSeverityNone)
}

// ErrorGroup returns the condition group with status False and severity Error, if any.
func (g conditionGroups) ErrorGroup() *conditionGroup {
	return g.getByStatusAndSeverity(metav1.ConditionFalse, kmapi.ConditionSeverityError)
}

// WarningGroup returns the condition group with status False and severity Warning, if any.
func (g conditionGroups) WarningGroup() *conditionGroup {
	return g.getByStatusAndSeverity(metav1.ConditionFalse, kmapi.ConditionSeverityWarning)
}

func (g conditionGroups) getByStatusAndSeverity(status metav1.ConditionStatus, severity kmapi.ConditionSeverity) *conditionGroup {
	if len(g) == 0 {
		return nil
	}
	for _, group := range g {
		if group.status == status && group.severity == severity {
			return &group
		}
	}
	return nil
}

// conditionGroup define a group of conditions with the same status and severity,
// and thus with the same priority when merging into a Ready condition.
type conditionGroup struct {
	status     metav1.ConditionStatus
	severity   kmapi.ConditionSeverity
	conditions []localizedCondition
}

// mergePriority provides a priority value for the status and severity tuple that identifies this
// condition group. The mergePriority value allows an easier sorting of conditions groups.
func (g conditionGroup) mergePriority() int {
	switch g.status {
	case metav1.ConditionFalse:
		switch g.severity {
		case kmapi.ConditionSeverityError:
			return 0
		case kmapi.ConditionSeverityWarning:
			return 1
		case kmapi.ConditionSeverityInfo:
			return 2
		}
	case metav1.ConditionTrue:
		return 3
	case metav1.ConditionUnknown:
		return 4
	}

	// this should never happen
	return 99
}
