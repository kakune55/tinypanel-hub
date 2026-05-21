package httpapi

import "tinypanel-hub/internal/domain"

func publicUser(user domain.User) domain.User {
	user.APITokenHash = ""
	return user
}

func publicDevice(device domain.Device) domain.Device {
	device.SecretHash = ""
	return device
}

func publicDevices(devices []domain.Device) []domain.Device {
	out := make([]domain.Device, len(devices))
	for i, device := range devices {
		out[i] = publicDevice(device)
	}
	return out
}
