package result

import (
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// todo: In the near future it will be merged into the main logic
func requeueIfError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

func noRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func requeueAfter(interval time.Duration, err error) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: interval}, err
}

func requeueImmediately() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}
