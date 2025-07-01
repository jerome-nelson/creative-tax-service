package shared

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type JiraConfig struct {
	Cid         string
	RedirectUrl string
	Secret      string
	OauthUrl    string
}

type Oauth struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	// Requires the scope `offline_access` to be set
	RefreshToken string `json:"refresh_token"`
}

// TODO: Nedds stronger types
type OauthScopes = []string

func SetAuthUrl(config JiraConfig) string {
	scopes := OauthScopes{
		"offline_access",
		"read:me",
		"read:project.avatar:jira",
		"read:filter:jira",
		"read:group:jira",
		"read:issue:jira",
		"read:attachment:jira",
		"read:comment:jira",
		"read:comment.property:jira",
		"read:field:jira",
		"read:issue-details:jira",
		"read:field.default-value:jira",
		"read:field.option:jira",
		"read:field:jira",
		"read:group:jira",
	}

	baseURL := "https://auth.atlassian.com/authorize"
	params := url.Values{}
	params.Set("audience", "api.atlassian.com")
	params.Set("client_id", config.Cid) // fill your actual client_id
	params.Set("redirect_uri", config.RedirectUrl)
	params.Set("response_type", "code")
	params.Set("prompt", "consent")

	params.Set("scope", strings.Join(scopes, " "))

	return fmt.Sprintf("%s?%s", baseURL, params.Encode())
}

func AuthGuard(log *log.Logger) func(h http.HandlerFunc) http.HandlerFunc {
	log.Println("authGuard init")
	return func(h http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var isAuthed bool = false
			for _, cookie := range r.Cookies() {
				if cookie.Name == "oauth_token" {
					isAuthed = true
				}
			}
			if isAuthed != true {
				log.Printf("attempted to access auth route %s\n", r.URL.Path)
				http.Error(w, "Not authorised", http.StatusUnauthorized)
				return
			}

			h(w, r)
		}
	}
}

// SetJiraCookies
// Sets Jira Secure cookies for client. Must have a buffer of at least 61 seconds.
func SetJiraCookie(w http.ResponseWriter, log *log.Logger, details Oauth) {
	offsetExpiry := details.ExpiresIn - 60
	if offsetExpiry < 0 != false {
		log.Println("attempted to set cookie with negative expiry.")
		http.Error(w, "Cookie cannot be set", http.StatusBadRequest)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_token",
		Value:    details.AccessToken,
		Path:     "/",
		MaxAge:   offsetExpiry,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "scopes",
		Value:    details.Scope,
		Path:     "/",
		MaxAge:   offsetExpiry,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    details.RefreshToken,
		Path:     "/",
		MaxAge:   offsetExpiry,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	now := time.Now().UTC()
	futureTime := now.Add(time.Duration(offsetExpiry) * time.Second)

	http.SetCookie(w, &http.Cookie{
		Name:     "expiry",
		Value:    futureTime.Format("2006-01-02T15:04:05.000Z"),
		Path:     "/",
		MaxAge:   offsetExpiry,
		HttpOnly: false,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	if err := Encode(w, http.StatusOK, map[string]string{
		"scope": details.Scope,
		"type":  details.TokenType,
	}); err != nil {
		log.Println(err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
