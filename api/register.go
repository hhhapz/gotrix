package api

import (
	"encoding/json"
	"errors"

	"github.com/chanbakjsd/gomatrix/matrix"
)

// Errors returned by (*Client).Register().
// It may also be returned by any auth functions in InteractiveRegister.
var (
	ErrUserIDTaken          = errors.New("requested user ID has already been taken")
	ErrMalformedUserID      = errors.New("invalid characters found in user ID")
	ErrReservedUserID       = errors.New("the user ID has been reserved for other purposes")
	ErrRegistrationDisabled = errors.New("registration for the specified user type has been disabled")
)

// RegisterArg represents arguments for the Register function.
type RegisterArg struct {
	Auth                     interface{} `json:"auth,omitempty"`
	Username                 string      `json:"username"`
	Password                 string      `json:"password"`
	DeviceID                 string      `json:"device_id,omitempty"`
	InitialDeviceDisplayName string      `json:"initial_device_display_name,omitempty"`
	InhibitLogin             bool        `json:"inhibit_login,omitempty"`
}

// RegisterResponse represents the success response from the register endpoint.
type RegisterResponse struct {
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
	DeviceID    string `json:"device_id"`
}

// Register registers an account on the homeserver with the provided arguments.
// Once the authentication is successful, the client is automatically logged in
// if InhibitLogin is set to false in RegisterArg.
//
// It returns an InteractiveRegister object which implements the User Interactive
// Authentication API.
// Users may choose to call InteractiveRegister.RegisterResponse() to inspect the
// RegisterResponse.
//
// It implements the `POST _matrix/client/r0/register` endpoint.
func (c *Client) Register(kind string, req RegisterArg) (InteractiveRegister, error) {
	ir := InteractiveRegister{
		UserInteractiveAuthAPI: &UserInteractiveAuthAPI{},
	}

	ir.Request = func(auth, to interface{}) error {
		req.Auth = auth
		return c.Request(
			"POST", "_matrix/client/r0/register", to,
			map[matrix.ErrorCode]error{
				matrix.CodeUserInUse:       ErrUserIDTaken,
				matrix.CodeInvalidUsername: ErrMalformedUserID,
				matrix.CodeExclusive:       ErrReservedUserID,
				matrix.CodeForbidden:       ErrRegistrationDisabled,
			},
			WithQuery(map[string]string{
				"kind": kind,
			}),
			WithBody(req),
		)
	}

	ir.SuccessCallback = func(json.RawMessage) error {
		resp, err := ir.RegisterResponse()
		if err != nil {
			return err
		}
		// If inhibitLogin is set, the homeserver probably does not supply us
		// with the info we want.
		if !req.InhibitLogin {
			c.UserID = resp.UserID
			c.AccessToken = resp.AccessToken
			c.DeviceID = resp.DeviceID
		}
		return nil
	}
	return ir, nil
}

// InteractiveRegister is a struct that adds helper functions onto UserInteractiveAuthAPI.
// To see functions on authenticating, refer to it instead.
type InteractiveRegister struct {
	*UserInteractiveAuthAPI
}

// RegisterResponse formats the Result() as a RegisterResponse.
//
// It returns an error if there isn't any result yet.
func (i InteractiveRegister) RegisterResponse() (*RegisterResponse, error) {
	msg, err := i.Result()
	if err != nil {
		return nil, err
	}

	var result *RegisterResponse
	err = json.Unmarshal(*msg, result)
	return result, err
}