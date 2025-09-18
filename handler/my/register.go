package my

import "amway/handler"

// RegisterHandlers registers all handlers for the "My Amway" feature.
func RegisterHandlers() {
	handler.AddComponentHandler("my_amway_button", MyAmwayButtonHandler)
	handler.AddComponentHandlerPrefix("my_amway_page", MyAmwayPageHandler)

	// New handlers for modification flow
	handler.AddComponentHandlerPrefix("modify_amway_button", ModifyAmwayButtonHandler)
	handler.AddModalHandler("modify_amway_modal", ModifyAmwayModalHandler)
	handler.AddComponentHandlerPrefix("retract_post_button", RetractPostHandler)
	handler.AddComponentHandlerPrefix("toggle_anonymity_button", ToggleAnonymityHandler)
	handler.AddComponentHandlerPrefix("delete_amway_button", DeleteAmwayHandler)
	handler.AddComponentHandlerPrefix("back_to_my_amway", BackToMyAmwayHandler)
}
