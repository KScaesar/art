package Artifex

import (
	"testing"
)

func BenchmarkHandleMessage(b *testing.B) {
	mux := NewMux("/")

	b.StopTimer()
	for i, subject := range githubAPI {
		_ = i
		// fmt.Printf("%v api %v\n", i, subject)
		mux.Handler(subject, UseSkipMessage())
	}

	b.ReportAllocs()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, subject := range githubAPI {

			// cpu: 13th Gen Intel(R) Core(TM) i5-1340P
			// BenchmarkHandleMessage
			// BenchmarkHandleMessage-16    	   10000	    152405 ns/op	   79005 B/op	     825 allocs/op
			//
			// message := newMessage()

			// cpu: 13th Gen Intel(R) Core(TM) i5-1340P
			// BenchmarkHandleMessage
			// BenchmarkHandleMessage-16    	   20233	     58824 ns/op	    3954 B/op	     247 allocs/op
			//
			message := GetMessage()

			message.Subject = subject
			err := mux.HandleMessage(message, nil)
			PutMessage(message)
			if err != nil {
				b.Errorf("Error handling message: %v", err)
			}
		}
	}
}

// https://github.com/julienschmidt/go-http-routing-benchmark?tab=readme-ov-file
var githubAPI = []string{
	"/authorizations",
	"/authorizations/{id}",
	"/authorizations/clients/{client_id}",
	"/applications/{client_id}/tokens/{access_token}",
	"/applications/{client_id}/tokens",

	"/events",
	"/repos/{owner}/{repo}/events",
	"/networks/{owner}/{repo}/events",
	"/orgs/{org}/events",
	"/users/{user}/received_events",
	"/users/{user}/received_events/public",
	"/users/{user}/events",
	"/users/{user}/events/public",
	"/users/{user}/events/orgs/{org}",
	"/feeds",
	"/notifications",
	"/repos/{owner}/{repo}/notifications",
	"/notifications/threads/{id}",
	"/notifications/threads/{id}/subscription",
	"/repos/{owner}/{repo}/stargazers",
	"/users/{user}/starred",
	"/user/starred",
	"/user/starred/{owner}/{repo}",
	"/repos/{owner}/{repo}/subscribers",
	"/users/{user}/subscriptions",
	"/user/subscriptions",
	"/repos/{owner}/{repo}/subscription",
	"/user/subscriptions/{owner}/{repo}",

	"/users/{user}/gists",
	"/gists",
	"/gists/public",
	"/gists/starred",
	"/gists/{id}",
	"/gists/{id}/star",
	"/gists/{id}/forks",

	"/repos/{owner}/{repo}/git/blobs/{sha}",
	"/repos/{owner}/{repo}/git/blobs",
	"/repos/{owner}/{repo}/git/commits/{sha}",
	"/repos/{owner}/{repo}/git/commits",
	"/repos/{owner}/{repo}/git/refs/*ref",
	"/repos/{owner}/{repo}/git/refs",
	"/repos/{owner}/{repo}/git/tags/{sha}",
	"/repos/{owner}/{repo}/git/tags",
	"/repos/{owner}/{repo}/git/trees/{sha}",
	"/repos/{owner}/{repo}/git/trees",

	"/issues",
	"/user/issues",
	"/orgs/{org}/issues",
	"/repos/{owner}/{repo}/issues",
	"/repos/{owner}/{repo}/issues/{number}",
	"/repos/{owner}/{repo}/assignees",
	"/repos/{owner}/{repo}/assignees/{assignee}",
	"/repos/{owner}/{repo}/issues/{number}/comments",
	"/repos/{owner}/{repo}/issues/comments",
	"/repos/{owner}/{repo}/issues/comments/{id}",
	"/repos/{owner}/{repo}/issues/{number}/events",
	"/repos/{owner}/{repo}/issues/events",
	"/repos/{owner}/{repo}/issues/events/{id}",
	"/repos/{owner}/{repo}/labels",
	"/repos/{owner}/{repo}/labels/{name}",
	"/repos/{owner}/{repo}/issues/{number}/labels",
	"/repos/{owner}/{repo}/issues/{number}/labels/{name}",
	"/repos/{owner}/{repo}/milestones/{number}/labels",
	"/repos/{owner}/{repo}/milestones",
	"/repos/{owner}/{repo}/milestones/{number}",

	"/emojis",
	"/gitignore/templates",
	"/gitignore/templates/{name}",
	"/markdown",
	"/markdown/raw",
	"/meta",
	"/rate_limit",

	"/users/{user}/orgs",
	"/user/orgs",
	"/orgs/{org}",
	"/orgs/{org}/members",
	"/orgs/{org}/members/{user}",
	"/orgs/{org}/public_members",
	"/orgs/{org}/public_members/{user}",
	"/orgs/{org}/teams",
	"/teams/{id}",
	"/teams/{id}/members",
	"/teams/{id}/members/{user}",
	"/teams/{id}/repos",
	"/teams/{id}/repos/{owner}/{repo}",
	"/user/teams",

	"/repos/{owner}/{repo}/pulls",
	"/repos/{owner}/{repo}/pulls/{number}",
	"/repos/{owner}/{repo}/pulls/{number}/commits",
	"/repos/{owner}/{repo}/pulls/{number}/files",
	"/repos/{owner}/{repo}/pulls/{number}/merge",
	"/repos/{owner}/{repo}/pulls/{number}/comments",
	"/repos/{owner}/{repo}/pulls/comments",
	"/repos/{owner}/{repo}/pulls/comments/{number}",

	"/user/repos",
	"/users/{user}/repos",
	"/orgs/{org}/repos",
	"/repositories",
	"/repos/{owner}/{repo}",
	"/repos/{owner}/{repo}/contributors",
	"/repos/{owner}/{repo}/languages",
	"/repos/{owner}/{repo}/teams",
	"/repos/{owner}/{repo}/tags",
	"/repos/{owner}/{repo}/branches",
	"/repos/{owner}/{repo}/branches/{branch}",
	"/repos/{owner}/{repo}/collaborators",
	"/repos/{owner}/{repo}/collaborators/{user}",
	"/repos/{owner}/{repo}/comments",
	"/repos/{owner}/{repo}/commits/{sha}/comments",
	"/repos/{owner}/{repo}/comments/{id}",
	"/repos/{owner}/{repo}/commits",
	"/repos/{owner}/{repo}/commits/{sha}",
	"/repos/{owner}/{repo}/readme",
	"/repos/{owner}/{repo}/contents/*path",
	"/repos/{owner}/{repo}/{archive_format}/{ref}",
	"/repos/{owner}/{repo}/keys",
	"/repos/{owner}/{repo}/keys/{id}",
	"/repos/{owner}/{repo}/downloads",
	"/repos/{owner}/{repo}/downloads/{id}",
	"/repos/{owner}/{repo}/forks",
	"/repos/{owner}/{repo}/hooks",
	"/repos/{owner}/{repo}/hooks/{id}",
	"/repos/{owner}/{repo}/hooks/{id}/tests",
	"/repos/{owner}/{repo}/merges",
	"/repos/{owner}/{repo}/releases",
	"/repos/{owner}/{repo}/releases/{id}",
	"/repos/{owner}/{repo}/releases/{id}/assets",
	"/repos/{owner}/{repo}/stats/contributors",
	"/repos/{owner}/{repo}/stats/commit_activity",
	"/repos/{owner}/{repo}/stats/code_frequency",
	"/repos/{owner}/{repo}/stats/participation",
	"/repos/{owner}/{repo}/stats/punch_card",
	"/repos/{owner}/{repo}/statuses/{ref}",

	"/search/repositories",
	"/search/code",
	"/search/issues",
	"/search/users",
	"/legacy/issues/search/{owner}/{repository}/{state}/{keyword}",
	"/legacy/repos/search/{keyword}",
	"/legacy/user/search/{keyword}",
	"/legacy/user/email/{email}",

	"/users/{user}",
	"/user",
	"/users",
	"/user/emails",
	"/users/{user}/followers",
	"/user/followers",
	"/user/following/{user}",
	"/users/{user}/following/{target_user}",
	"/users/{user}/keys",
	"/user/keys",
	"/user/keys/{id}",
}
