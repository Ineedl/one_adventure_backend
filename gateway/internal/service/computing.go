package service

import ()

type (
	IComputingService interface {
	}
)

var (
	localComputingService IComputingService
)

func ComputingService() IComputingService {
	if localComputingService == nil {
		panic("implement not found for interface IComputing, forgot register?")
	}
	return localComputingService
}

func RegisterLoginComputingService(i IComputingService) {
	localComputingService = i
}
