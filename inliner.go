package premailer

type Inliner func(string) (string, error)

func NaiveInliner(raw string) (string, error) {
	return raw, nil
}
