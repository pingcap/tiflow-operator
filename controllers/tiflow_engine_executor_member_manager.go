package controllers

import (
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type ExecutorMember struct {
}

func NewExecutorMemberManager() {

}

func getExecutorConfigMap() (*corev1.ConfigMap, error) {

	return nil, nil
}

func (m *ExecutorMember) Sync() error {
	return nil
}
func (m *ExecutorMember) syncExecutorConfigMap() (*corev1.ConfigMap, error) {
	return nil, nil
}

func (m *ExecutorMember) syncStatefulSet() error {
	return nil
}

func (m *ExecutorMember) syncExecutorStatus() error {
	return nil
}

func (m *ExecutorMember) syncExecutorHeadlessService() error {
	return nil
}

func getNewExecutorHeadlessService() *corev1.Service {
	return nil
}

func (m ExecutorMember) getNewExecutorStatefulSet() (*apps.StatefulSet, error) {
	return nil, nil
}
