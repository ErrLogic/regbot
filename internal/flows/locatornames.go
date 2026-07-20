package flows

// InstagramLocatorNames are the element names the Instagram flow depends on.
// Order mirrors the verified signup wizard: welcome → email → OTP → password →
// birthday → name → username → terms → post-signup interstitials.
var InstagramLocatorNames = []string{
	"create_new_account",
	"switch_to_email",
	"email_field",
	"next_button",
	"confirmation_code_field",
	"confirm_code_button",
	"password_field",
	"birthday_year_picker",
	"birthday_set",
	"birthday_next",
	"full_name_field",
	"username_field",
	"username_taken_error",
	"agree_terms_button",
	"skip_button",
	"not_now_button",
	"finish_button",
}

// TikTokLocatorNames are the element names the TikTok flow depends on.
var TikTokLocatorNames = []string{
	"dismiss_sheet",
	"agree_terms",
	"allow_button",
	"login_button",
	"sign_up_button",
	"sso_continue",
	"sso_account_row",
	"use_phone_or_email",
	"email_tab",
	"birthday_next",
	"email_field",
	"send_code_button",
	"code_field",
	"next_button",
	"password_field",
	"nickname_field",
	"skip_button",
	"finish_button",
}
