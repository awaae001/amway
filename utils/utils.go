package utils

// StringPtr returns a pointer to the given string.
// This is a helper function for discordgo fields that require a *string.
func StringPtr(s string) *string {
	return &s
}
