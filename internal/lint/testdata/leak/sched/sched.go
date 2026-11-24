package sched

import "fixture/queue"

func Admit(q *queue.RunQueue) int {
	q.Items = append(q.Items, 1)
	return q.Head
}
