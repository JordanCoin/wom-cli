package cmd

import (
	"fmt"
	"os"
)

// The two values a group competition needs that nobody wants on a command
// line.
//
// A group's verification code is a long-lived secret: it authorises editing
// and deleting every competition the group owns, and it does not rotate on
// its own. Anything that shells out to `wom` would otherwise have to put it
// in argv, where it shows up in `ps`, in shell history, and in whatever log
// the caller keeps of the commands it ran. An environment variable is read
// by the child process and by nothing else.
//
// The group id is not a secret (5165 is on the group's public page); it
// falls back the same way only so a caller that has already set one for a
// clan does not have to repeat it on every invocation.
const (
	envVerificationCode = "WOM_VERIFICATION_CODE"
	envGroupID          = "WOM_GROUP_ID"
)

// resolveSecret takes the flag when it was given and falls back to the
// environment otherwise.
//
// The flag wins on purpose. The environment is the ambient default a shell
// profile or a service unit sets once; an explicit flag is somebody saying
// "not that one, this one" about a single invocation. A flag that lost to
// the environment would leave no way to say it.
func resolveSecret(flagValue, envKey string) string {
	if flagValue != "" {
		return flagValue
	}
	return os.Getenv(envKey)
}

// missingCodeError is the same refusal everywhere a verification code is
// required, so no command tells a caller about only one of the two ways to
// supply one.
func missingCodeError() error {
	return fmt.Errorf("a verification code is required: pass --verification-code or set %s", envVerificationCode)
}

// verificationCodeFlagHelp and groupIDFlagHelp are shared so the fallback is
// documented identically on every command that has the flag.
const (
	verificationCodeFlagHelp = "Group verification code. Falls back to $" + envVerificationCode + ", which keeps the secret out of argv"
	groupIDFlagHelp          = "Group ID, for group competitions. Falls back to $" + envGroupID
)
