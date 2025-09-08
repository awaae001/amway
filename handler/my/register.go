package my

import "amway/handler"

// RegisterHandlers registers all handlers for the "My Amway" feature.
func RegisterHandlers() {
	handler.AddComponentHandler("my_amway_button", MyAmwayButtonHandler)
	handler.AddComponentHandlerPrefix("my_amway_page", MyAmwayPageHandler)
	handler.AddComponentHandlerPrefix("retract_submission_button", RetractSubmissionButtonHandler)
	handler.AddComponentHandlerPrefix("retract_submission_modal", RetractSubmissionModalHandler)
}
