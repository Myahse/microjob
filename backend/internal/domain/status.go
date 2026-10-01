// Package domain holds the pure business rules of MicroJob: the job
// lifecycle, commission maths, distances and matching scores.
package domain

import "slices"

// JobStatus is the lifecycle state of a job (cahier des charges §16).
type JobStatus string

const (
	StatusPosted               JobStatus = "POSTED"
	StatusMatching             JobStatus = "MATCHING"
	StatusAssigned             JobStatus = "ASSIGNED"
	StatusWorkerOnTheWay       JobStatus = "WORKER_ON_THE_WAY"
	StatusArrived              JobStatus = "ARRIVED"
	StatusStarted              JobStatus = "STARTED"
	StatusAwaitingConfirmation JobStatus = "AWAITING_CONFIRMATION"
	StatusCompleted            JobStatus = "COMPLETED"
	StatusPaid                 JobStatus = "PAID"
	StatusRated                JobStatus = "RATED"
	StatusCancelled            JobStatus = "CANCELLED"
	StatusDisputed             JobStatus = "DISPUTED"
)

// Actor identifies who drives a transition.
type Actor string

const (
	ActorClient Actor = "client"
	ActorWorker Actor = "worker"
	ActorSystem Actor = "system"
)

type transition struct {
	from  []JobStatus
	actor Actor
}

// workerSteps are the progress actions the assigned worker performs.
var workerSteps = map[JobStatus]transition{
	StatusWorkerOnTheWay:       {from: []JobStatus{StatusAssigned}, actor: ActorWorker},
	StatusArrived:              {from: []JobStatus{StatusAssigned, StatusWorkerOnTheWay}, actor: ActorWorker},
	StatusStarted:              {from: []JobStatus{StatusAssigned, StatusWorkerOnTheWay, StatusArrived}, actor: ActorWorker},
	StatusAwaitingConfirmation: {from: []JobStatus{StatusStarted}, actor: ActorWorker},
}

// CanAdvance reports whether actor may move a job from `from` to `to`
// through the regular progress endpoint.
func CanAdvance(from, to JobStatus, actor Actor) bool {
	t, ok := workerSteps[to]
	return ok && t.actor == actor && slices.Contains(t.from, from)
}

// IsOpen reports whether a job can still receive a worker.
func IsOpen(s JobStatus) bool {
	return s == StatusPosted || s == StatusMatching
}

// IsActive reports whether a worker is assigned and the job is in progress.
func IsActive(s JobStatus) bool {
	switch s {
	case StatusAssigned, StatusWorkerOnTheWay, StatusArrived, StatusStarted, StatusAwaitingConfirmation:
		return true
	}
	return false
}

// IsFinished reports whether the work is done and confirmed.
func IsFinished(s JobStatus) bool {
	return s == StatusCompleted || s == StatusPaid || s == StatusRated
}

// CanCancel reports whether actor may cancel a job in status s.
// Clients can cancel before the work starts; workers can drop an assigned
// job before starting it (the job then returns to matching).
func CanCancel(s JobStatus, actor Actor) bool {
	switch actor {
	case ActorClient:
		return IsOpen(s) || s == StatusAssigned || s == StatusWorkerOnTheWay || s == StatusArrived
	case ActorWorker:
		return s == StatusAssigned || s == StatusWorkerOnTheWay || s == StatusArrived
	}
	return false
}

// CancellationHurtsReputation reports whether a cancellation counts against
// the actor. Cancelling an open job is free; dropping someone who is already
// committed (worker on the way, client waiting) is not.
func CancellationHurtsReputation(s JobStatus, actor Actor) bool {
	if actor == ActorWorker {
		return true
	}
	return s == StatusWorkerOnTheWay || s == StatusArrived
}

// CanReport reports whether a dispute may be opened in status s.
func CanReport(s JobStatus) bool {
	return IsActive(s) || IsFinished(s)
}

// AddressVisible reports whether the exact address may be shown to a viewer.
func AddressVisible(isClient, isAssignedWorker bool) bool {
	return isClient || isAssignedWorker
}
