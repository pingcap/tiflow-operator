package result

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

const (
	ShortPauseTime = 5 * time.Second
	LongPauseTime  = 10 * time.Second
)

func RequeueIfError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func NoRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func RequeueAfter(interval time.Duration, err error) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: interval}, err
}

func RequeueImmediately() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}
