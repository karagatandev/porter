package kubernetes

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"syscall"

	"github.com/pkg/errors"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

type K8sConnectionError interface {
	Externalize() error
	Error() string
}

func CatchK8sConnectionError(err error) K8sConnectionError {
	if uerr, ok := err.(*url.Error); ok {
		if noerr, ok := uerr.Err.(*net.OpError); ok {
			if scerr, ok := noerr.Err.(*os.SyscallError); ok {
				if scerr.Err == syscall.ECONNREFUSED {
					return &ErrConnection{
						k8sErr: err,
					}
				}
			}
		}
	}

	if k8sErrors.IsTimeout(err) {
		return &ErrConnection{
			k8sErr: err,
		}
	}

	if k8sErrors.IsUnauthorized(err) || k8sErrors.IsForbidden(err) {
		return &ErrUnauthorized{
			k8sErr: err,
		}
	}

	return &ErrUnknown{
		k8sErr: err,
	}
}

type ErrUnknown struct {
	k8sErr error
}

func (e *ErrUnknown) Error() string {
	return fmt.Sprintf("Unknown or unhandled error: %s", e.k8sErr.Error())
}

func (e *ErrUnknown) Externalize() error {
	return errors.Errorf("Unknown or unhandled error: %v", e.k8sErr)
}

// For ECONNREFUSED and errors.IsTimeout
type ErrConnection struct {
	k8sErr error
}

func (e *ErrConnection) Error() string {
	return fmt.Sprintf("Could not connect to cluster: %s", e.k8sErr.Error())
}

func (e *ErrConnection) Externalize() error {
	return errors.Errorf("Could not connect to cluster: %v", e.k8sErr)
}

// For errors.IsForbidden and errors.IsUnauthorized
type ErrUnauthorized struct {
	k8sErr error
}

func (e *ErrUnauthorized) Error() string {
	return fmt.Sprintf("Unauthorized: %v", e.k8sErr)
}

func (e *ErrUnauthorized) Externalize() error {
	return errors.Errorf("Unauthorized: %v", e.k8sErr)
}
