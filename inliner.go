package premailer

import "errors"

type Inliner func(string) (string, error)

func NaiveInliner(raw string) (string, error) {
	return raw, errors.New("unimplemented")
}
