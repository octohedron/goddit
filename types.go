package main

// PopularSubreddits are the popular subreddits
type PopularSubreddits struct {
	Kind string `bson:"kind,omitempty" json:"kind"`
	Data struct {
		Modhash  string `bson:"modhash,omitempty" json:"modhash"`
		Dist     int    `bson:"dist,omitempty" json:"dist"`
		Children []struct {
			Kind string `bson:"kind,omitempty" json:"kind"`
			Data struct {
				UserFlairBackgroundColor   interface{}   `bson:"user_flair_background_color,omitempty" json:"user_flair_background_color"`
				SubmitTextHTML             interface{}   `bson:"submit_text_html,omitempty" json:"submit_text_html"`
				RestrictPosting            bool          `bson:"restrict_posting,omitempty" json:"restrict_posting"`
				UserIsBanned               bool          `bson:"user_is_banned,omitempty" json:"user_is_banned"`
				FreeFormReports            bool          `bson:"free_form_reports,omitempty" json:"free_form_reports"`
				WikiEnabled                interface{}   `bson:"wiki_enabled,omitempty" json:"wiki_enabled"`
				UserIsMuted                bool          `bson:"user_is_muted,omitempty" json:"user_is_muted"`
				UserCanFlairInSr           interface{}   `bson:"user_can_flair_in_sr,omitempty" json:"user_can_flair_in_sr"`
				DisplayName                string        `bson:"display_name,omitempty" json:"display_name"`
				HeaderImg                  interface{}   `bson:"header_img,omitempty" json:"header_img"`
				Title                      string        `bson:"title,omitempty" json:"title"`
				IconSize                   interface{}   `bson:"icon_size,omitempty" json:"icon_size"`
				PrimaryColor               string        `bson:"primary_color,omitempty" json:"primary_color"`
				ActiveUserCount            interface{}   `bson:"active_user_count,omitempty" json:"active_user_count"`
				IconImg                    string        `bson:"icon_img,omitempty" json:"icon_img"`
				DisplayNamePrefixed        string        `bson:"display_name_prefixed,omitempty" json:"display_name_prefixed"`
				AccountsActive             interface{}   `bson:"accounts_active,omitempty" json:"accounts_active"`
				PublicTraffic              bool          `bson:"public_traffic,omitempty" json:"public_traffic"`
				Subscribers                int           `bson:"subscribers,omitempty" json:"subscribers"`
				UserFlairRichtext          []interface{} `bson:"user_flair_richtext,omitempty" json:"user_flair_richtext"`
				VideostreamLinksCount      int           `bson:"videostream_links_count,omitempty" json:"videostream_links_count"`
				Name                       string        `bson:"name,omitempty" json:"name"`
				Quarantine                 bool          `bson:"quarantine,omitempty" json:"quarantine"`
				HideAds                    bool          `bson:"hide_ads,omitempty" json:"hide_ads"`
				EmojisEnabled              bool          `bson:"emojis_enabled,omitempty" json:"emojis_enabled"`
				AdvertiserCategory         string        `bson:"advertiser_category,omitempty" json:"advertiser_category"`
				PublicDescription          string        `bson:"public_description,omitempty" json:"public_description"`
				CommentScoreHideMins       int           `bson:"comment_score_hide_mins,omitempty" json:"comment_score_hide_mins"`
				UserHasFavorited           bool          `bson:"user_has_favorited,omitempty" json:"user_has_favorited"`
				UserFlairTemplateID        interface{}   `bson:"user_flair_template_id,omitempty" json:"user_flair_template_id"`
				CommunityIcon              string        `bson:"community_icon,omitempty" json:"community_icon"`
				BannerBackgroundImage      string        `bson:"banner_background_image,omitempty" json:"banner_background_image"`
				OriginalContentTagEnabled  bool          `bson:"original_content_tag_enabled,omitempty" json:"original_content_tag_enabled"`
				SubmitText                 string        `bson:"submit_text,omitempty" json:"submit_text"`
				DescriptionHTML            string        `bson:"description_html,omitempty" json:"description_html"`
				SpoilersEnabled            bool          `bson:"spoilers_enabled,omitempty" json:"spoilers_enabled"`
				HeaderTitle                interface{}   `bson:"header_title,omitempty" json:"header_title"`
				HeaderSize                 interface{}   `bson:"header_size,omitempty" json:"header_size"`
				UserFlairPosition          string        `bson:"user_flair_position,omitempty" json:"user_flair_position"`
				AllOriginalContent         bool          `bson:"all_original_content,omitempty" json:"all_original_content"`
				HasMenuWidget              bool          `bson:"has_menu_widget,omitempty" json:"has_menu_widget"`
				IsEnrolledInNewModmail     interface{}   `bson:"is_enrolled_in_new_modmail,omitempty" json:"is_enrolled_in_new_modmail"`
				KeyColor                   string        `bson:"key_color,omitempty" json:"key_color"`
				CanAssignUserFlair         bool          `bson:"can_assign_user_flair,omitempty" json:"can_assign_user_flair"`
				Created                    float64       `bson:"created,omitempty" json:"created"`
				Wls                        interface{}   `bson:"wls,omitempty" json:"wls"`
				ShowMediaPreview           bool          `bson:"show_media_preview,omitempty" json:"show_media_preview"`
				SubmissionType             string        `bson:"submission_type,omitempty" json:"submission_type"`
				UserIsSubscriber           bool          `bson:"user_is_subscriber,omitempty" json:"user_is_subscriber"`
				DisableContributorRequests bool          `bson:"disable_contributor_requests,omitempty" json:"disable_contributor_requests"`
				AllowVideogifs             bool          `bson:"allow_videogifs,omitempty" json:"allow_videogifs"`
				UserFlairType              string        `bson:"user_flair_type,omitempty" json:"user_flair_type"`
				AllowPolls                 bool          `bson:"allow_polls,omitempty" json:"allow_polls"`
				CollapseDeletedComments    bool          `bson:"collapse_deleted_comments,omitempty" json:"collapse_deleted_comments"`
				EmojisCustomSize           interface{}   `bson:"emojis_custom_size,omitempty" json:"emojis_custom_size"`
				PublicDescriptionHTML      interface{}   `bson:"public_description_html,omitempty" json:"public_description_html"`
				AllowVideos                bool          `bson:"allow_videos,omitempty" json:"allow_videos"`
				IsCrosspostableSubreddit   bool          `bson:"is_crosspostable_subreddit,omitempty" json:"is_crosspostable_subreddit"`
				SuggestedCommentSort       interface{}   `bson:"suggested_comment_sort,omitempty" json:"suggested_comment_sort"`
				CanAssignLinkFlair         bool          `bson:"can_assign_link_flair,omitempty" json:"can_assign_link_flair"`
				AccountsActiveIsFuzzed     bool          `bson:"accounts_active_is_fuzzed,omitempty" json:"accounts_active_is_fuzzed"`
				SubmitTextLabel            interface{}   `bson:"submit_text_label,omitempty" json:"submit_text_label"`
				LinkFlairPosition          string        `bson:"link_flair_position,omitempty" json:"link_flair_position"`
				UserSrFlairEnabled         interface{}   `bson:"user_sr_flair_enabled,omitempty" json:"user_sr_flair_enabled"`
				UserFlairEnabledInSr       bool          `bson:"user_flair_enabled_in_sr,omitempty" json:"user_flair_enabled_in_sr"`
				AllowDiscovery             bool          `bson:"allow_discovery,omitempty" json:"allow_discovery"`
				UserSrThemeEnabled         bool          `bson:"user_sr_theme_enabled,omitempty" json:"user_sr_theme_enabled"`
				LinkFlairEnabled           bool          `bson:"link_flair_enabled,omitempty" json:"link_flair_enabled"`
				SubredditType              string        `bson:"subreddit_type,omitempty" json:"subreddit_type"`
				NotificationLevel          interface{}   `bson:"notification_level,omitempty" json:"notification_level"`
				BannerImg                  string        `bson:"banner_img,omitempty" json:"banner_img"`
				UserFlairText              interface{}   `bson:"user_flair_text,omitempty" json:"user_flair_text"`
				BannerBackgroundColor      string        `bson:"banner_background_color,omitempty" json:"banner_background_color"`
				ShowMedia                  bool          `bson:"show_media,omitempty" json:"show_media"`
				ID                         string        `bson:"id,omitempty" json:"id"`
				UserIsContributor          bool          `bson:"user_is_contributor,omitempty" json:"user_is_contributor"`
				Over18                     bool          `bson:"over18,omitempty" json:"over18"`
				Description                string        `bson:"description,omitempty" json:"description"`
				SubmitLinkLabel            interface{}   `bson:"submit_link_label,omitempty" json:"submit_link_label"`
				UserFlairTextColor         interface{}   `bson:"user_flair_text_color,omitempty" json:"user_flair_text_color"`
				RestrictCommenting         bool          `bson:"restrict_commenting,omitempty" json:"restrict_commenting"`
				UserFlairCSSClass          interface{}   `bson:"user_flair_css_class,omitempty" json:"user_flair_css_class"`
				AllowImages                bool          `bson:"allow_images,omitempty" json:"allow_images"`
				Lang                       string        `bson:"lang,omitempty" json:"lang"`
				WhitelistStatus            interface{}   `bson:"whitelist_status,omitempty" json:"whitelist_status"`
				URL                        string        `bson:"url,omitempty" json:"url"`
				CreatedUtc                 float64       `bson:"created_utc,omitempty" json:"created_utc"`
				BannerSize                 interface{}   `bson:"banner_size,omitempty" json:"banner_size"`
				MobileBannerImage          string        `bson:"mobile_banner_image,omitempty" json:"mobile_banner_image"`
				UserIsModerator            bool          `bson:"user_is_moderator,omitempty" json:"user_is_moderator"`
			}
		}
	}
}
