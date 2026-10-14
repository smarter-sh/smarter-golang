package smarter

// AccountsService holds the operations of Accounts: /api/v1/accounts/, and the manifest api's
// Account kind, with the account's users, contacts and user profiles. Only superusers may read
// them: the platform refuses other users with a 403. Users and accounts are created and changed with
// manifests (the User and Account kinds).
type AccountsService struct {
	*KindService[Account]
	// Users are the users of the api key's account: /api/v1/accounts/users/.
	Users *ReadService[User]
	// Contacts are the contacts of the api key's account: /api/v1/accounts/contacts/.
	Contacts *ReadService[AccountContact]
	// UserProfiles are each user's membership of an account: /api/v1/accounts/user-profiles/.
	UserProfiles *ReadService[UserProfile]
}
