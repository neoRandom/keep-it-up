- [x] TODO: add a username index on the players table, because username lookups are used for authentication and player lookup queries such as GetPlayerByUsername and password validation checks.

- [x] TODO: add GeneratePasswordHash to the Authentication interface. It should be used to generate a hashed password before sending password values to any query that expects a hashed password, especially in PlayerManagement.AddPlayer and PlayerManagement.UpdatePlayerPassword.
- [x] TODO: add CheckPlayerPassword to the Authentication interface. It should receive a password and a username, validate the credentials, and be used by login flows and the normal PlayerManagement password-update flow before allowing a password change.

- [ ] TODO: redesign the password change flow in the PlayerManagement interface and struct:
    - UpdatePlayerPassword: require the current/previous password before accepting a new password.
    - UpdatePlayerPasswordForce: add a force-update variant on the PlayerManagement interface that skips the previous-password check and is intended only for privileged or recovery flows.
    - BaseUpdatePlayerPassword: add a shared internal method on the PlayerManagement struct that contains the common validation and database update logic used by both password-update methods.
    - ValidatePlayerPassword: add this method to the Authentication interface so login and password-change flows can reuse the same password validation rule for checking the current password before sensitive mutations.
