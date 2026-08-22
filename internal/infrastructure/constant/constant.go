package constant

import "time"

const DBDatetimeFormat string = time.RFC3339

const EnvFilename string = ".env"

// SessionLifetime is the shared lifetime of an authentication session. It is used
// both for the login cookie expiry and the JWT exp claim so the two can never
// drift apart (see TODO 4).
const SessionLifetime time.Duration = 72 * time.Hour
