/*-
 * SPDX-License-Identifier: BSD-2-Clause
 *
 * Copyright (c) 2025 Brian J. Downs
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE REGENTS AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL THE REGENTS OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 */

#ifdef __cplusplus
extern "C" {
#endif
 
#ifndef __CLIENT_H
#define __CLIENT_H
 
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>
#include <sys/types.h>
 
#define GH_CLIENT_USER_BLOCKED_CODE     204
#define GH_CLIENT_USER_NOT_BLOCKED_CODE 404

#define GH_CLIENT_PER_PAGE_MAX 100

#define GH_API_BASE_URL   "https://api.github.com"
#define GH_API_ORGS_URL   GH_API_BASE_URL "/orgs"
#define GH_API_ORG_URL    GH_API_BASE_URL "/org"
#define GH_API_REPO_URL   GH_API_BASE_URL "/repos/"
#define GH_API_USER_URL   GH_API_BASE_URL "/user"
#define GH_API_USERS_URL  GH_API_BASE_URL "/users/"
#define GH_API_ISSUE_URL  GH_API_BASE_URL "/issue"
#define GH_API_ISSUES_URL GH_API_BASE_URL "/issues"

#define GH_MAX_URL_LEN 2048

/**
 * Contains the rate limit information returned from each API call.
 */
typedef struct {
    uint64_t limit;
    uint64_t remaining;
    uint64_t reset;
    uint64_t used;
    char *resource;
} gh_client_rate_limit_data_t;
 
/**
 * Default response structure returned for each call to the API. Contains the
 * API response, the response code, response size, any error code, and message
 * if applicable.
 */
typedef struct {
    char *resp;
    char *err_msg;
    size_t size;
    uint16_t resp_code;

    // pagination fields
    char first_link[GH_MAX_URL_LEN];
    char next_link[GH_MAX_URL_LEN];
    char prev_link[GH_MAX_URL_LEN];
    char last_link[GH_MAX_URL_LEN];

    // rate limit information
    gh_client_rate_limit_data_t *rate_limit_data;
} gh_client_response_t;

/**
 * Contains the states to choose from when listing objects.
 */
enum gh_item_list_state {
    GH_ITEM_STATE_OPENED = 0,
    GH_ITEM_STATE_CLOSED = 1,
    GH_ITEM_STATE_MERGED = 2,
    GH_ITEM_STATE_ALL    = 3
};

/**
 * Contains order options when listing objects.
 */
enum gh_item_list_order {
    GH_ORDER_DESC = 0,
    GH_ORDER_ASC  = 1
};

/**
 * gh_issue_filters
 */
enum gh_issue_filters {
    GH_ISSUE_FILTER_ASSIGNED   = 0, // default
    GH_ISSUE_FILTER_CREATED    = 1,
    GH_ISSUE_FILTER_MENTIONED  = 2,
    GH_ISSUE_FILTER_SUBSCRIBED = 3,
    GH_ISSUE_FILTER_REPOS      = 4,
    GH_ISSUE_FILTER_ALL        = 5
};

/**
 * gh_issue_sort_options
 */
enum gh_issue_sort_options {
    GH_ISSUE_SORT_CREATED  = 0, // default
    GH_ISSUE_SORT_UPDATED  = 1,
    GH_ISSUE_SORT_COMMENTS = 2
};

/**
 * Structure used to pass additional options when listing pull requests.
 */
typedef struct {
    enum gh_item_list_state state;
    enum gh_item_list_order order;
    unsigned int per_page;
    char page_url[GH_MAX_URL_LEN];
} gh_client_pull_req_opts_t;

/**
 * Structure used to pass additional options when listing issues.
 */
typedef struct {
    enum gh_item_list_state state;
    enum gh_item_list_order order;
    enum gh_issue_filters filter;
    enum gh_issue_sort_options sort;
    unsigned int per_page;
    char *assignee;
    char *creator;
    char *mention;
    char *labels;
    char *page_url;
    char *since; // expected format: YYYY-MM-DDTHH:MM:SSZ
    bool collab;
    bool orgs;
    bool owned;
    bool pulls;
} gh_client_issues_req_opts_t;

/**
 * Structure used to pass pagination settings.
 */
typedef struct {
    unsigned int per_page;
    char *page_url;
} gh_client_req_list_opts_t;

/**
 * Structure used to pass pagination settings.
 */
typedef struct {
    char *page_url;
    unsigned int first;
    unsigned int last;
    char *after;
    char *before;
    char *cursor;
} gh_client_req_list_cursor_opts_t;

/**
 * Structure used to pass additional options when listing commits.
 */
typedef struct {
    char *sha;
    char *path;
    char *author;
    char *committer;
    char *since;           // expected format: YYYY-MM-DDTHH:MM:SSZ
    char *until;           // expected format: YYYY-MM-DDTHH:MM:SSZ
    unsigned int per_page; // default: 30
    char *page_url;
} gh_client_commits_list_opts_t;

/**
 * Initialize the library.
 */
int
gh_client_init(const char *token);

/**
 * Set the value to be used as the user agent in requests.
 */
void
gh_client_set_user_agent(const char *ua);

/**
 * Free the memory used in the client response.
 */
void
gh_client_response_free(gh_client_response_t *res);

/**
 * Retrieve an octocat response giving an interesitng saying. The response
 * memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_octocat_says();

/**
 * Retrieve a list of releases for the given repository. The response memory
 * needs to be freed by the caller. 
 */
gh_client_response_t*
gh_client_repo_releases_list(const char *owner, const char *repo,
                            const gh_client_req_list_opts_t *opts);

/**
 * Retrieve the latest release for the given repository. The response memory
 * needs to be freed by the caller. 
 */
gh_client_response_t*
gh_client_repo_releases_latest(const char *owner, const char *repo);

/**
 * Retrieve a release by the given tag. The response memory needs to be freed
 * by the caller. 
 */
gh_client_response_t*
gh_client_repo_release_by_tag(const char *owner, const char *repo,
                            const char *tag);

/**
 * Retrieve a release by the given id. The response memory needs to be freed by
 * the caller. 
 */
gh_client_response_t*
gh_client_repo_release_by_id(const char *owner, const char *repo,
                            const unsigned int id);

/**
 * Create a new release for the given repository and configuration. The
 * response memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_repo_release_create(const char *owner, const char *repo,
                            const char *data);

/**
 * Update a release for the given repository and configuration. The response
 * memory needs to be freed by the caller.
 * 
 * data argument must be JSON in the following format:
 * 
 * {"tag_name":"v1.0.0","target_commitish":"master","name":"v1.0.0",
 *  "body":"Description of the release","draft":false,"prerelease":false}
 */
gh_client_response_t*
gh_client_repo_release_update(const char *owner, const char *repo,
                            const unsigned int id, const char *data);

/**
 * Delete a release for the given repository and configuration. The response
 * memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_repo_release_delete(const char *owner, const char *repo,
                            const unsigned int id);

/**
 * Generate release notes content for a release. The response memory needs to
 * be freed by the caller.
 * 
 * data argument must be JSON in the following format:
 * 
 * tag_name (required)
 * 
 * {"tag_name":"v1.0.0","target_commitish":"main","previous_tag_name":"v0.9.2",
 *  "configuration_file_path":".github/custom_release_config.yml"}
 */
gh_client_response_t*
gh_client_repo_release_gen_notes(const char *owner, const char *repo,
                                const char *data);

/**
 * List the assets on a release with the given. The response memory needs to be
 * freed by the caller.
 */
gh_client_response_t*
gh_client_repo_release_assets_list(const char *owner, const char *repo,
                                const unsigned int id,
                                const gh_client_req_list_opts_t *opts);

/**
 * Retrieve a release asset for th given id. The response memory needs to be
 * freed by the caller.
 */
gh_client_response_t*
gh_client_repo_release_asset_get(const char *owner, const char *repo,
                                const unsigned int id);

/**
 * Retrieve commits for a given repository. The response memory needs to be
 * freed by the caller.
 */
gh_client_response_t*
gh_client_repo_commits_list(const char *owner, const char *repo,
                            const gh_client_commits_list_opts_t *opts);

/**
 * Retrieve a single commit. The response memory needs to be freed by the
 * caller.
 */
gh_client_response_t*
gh_client_repo_commit_get(const char *owner, const char *repo,
                        const char *sha);

/**
 * Compare 2 commits. The response memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_repo_commits_compare(const char *owner, const char *repo,
                            const char *base, const char *head);

/**
 * Retrieve the merged pull request that introduced the commit. The response
 * memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_repo_pr_commits_list(const char *owner, const char *repo,
                            const char *sha,
                            const gh_client_req_list_opts_t *opts);

/**
 * Retrieve a list of branches for the given repository in JSON format. The
 * response memory needs to be freed by the caller. 
 */
gh_client_response_t*
gh_client_repo_branches_list(const char *owner, const char *repo,
                            const gh_client_req_list_opts_t *opts);

/**
 * Retrieve the given branch. The response memory needs to be freed by the
 * caller. 
 */
gh_client_response_t*
gh_client_repo_branch_get(const char *owner, const char *repo,
                        const char *branch);

/**
 * Rename the given branch. The response memory needs to be freed by the
 * caller.
 * 
 * data argument must be JSON in the following format:
 * '{"new_name":"my_renamed_branch"}'
 */
gh_client_response_t*
gh_client_repo_branch_rename(const char *owner, const char *repo,
                            const char *branch, const char *data);

/**
 * Sync the given branch in a fork to the given upstream. The response memory
 * needs to be freed by the caller. 
 * 
 * data argument must be JSON in the following format:
 * '{"branch":"<branch-name>"}'
 */
gh_client_response_t*
gh_client_repo_branch_sync_upstream(const char *owner, const char *repo,
                                    const char *branch, const char *data);

/**
 * Merge a branch. The response memory needs to be freed by the caller. 
 * 
 * data argument must be JSON in the following format:
 * '{"base":"master","head":"cool_feature",
 *   "commit_message":"Shipped cool_feature!"}'
 */
gh_client_response_t*
gh_client_repo_branch_merge(const char *owner, const char *repo,
                            const char *data);

/**
 * Retrieve a list of open pull requests. The response memory needs to be freed
 * by the caller.
 */
gh_client_response_t*
gh_client_repo_pull_request_list(const char *owner, const char *repo,
                                const gh_client_pull_req_opts_t *opts);

/**
 * Retrieve 1 pull request by id. order option in opts will be ignored. The
 * response memory needs to be freed by the caller. 
 */
gh_client_response_t*
gh_client_repo_pull_request_get(const char *owner, const char *repo,
                                const unsigned int id,
                                const gh_client_pull_req_opts_t *opts);

/**
 * Retrieve account information for the user currently logged in. The response
 * memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_user_logged_in_get();

/**
 * Retrieve account information for the given username. The response memory
 * needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_user_by_id_get(const char *username);

/**
 * Retrieve hovercard for the given username. The response memory needs to be
 * freed by the caller.
 */
gh_client_response_t*
gh_client_user_by_id_hovercard_get(const char *username);

/**
 * Retrieve a list of blocked users for the currently logged in user. The
 * response memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_user_blocked_list(const gh_client_req_list_opts_t *opts);

/**
 * Checks if the given username is blocked by the currently logged in user. If
 * the response code is 204, the given user is blocked but if the response code
 * is 404, the given user is not blocked. The response memory needs to be freed
 * by the caller.
 */
gh_client_response_t*
gh_client_user_blocked_by_id(const char *username);

/**
 * Blocks a user by the given id. The response memory needs to be freed by the
 * caller.
 */
gh_client_response_t*
gh_client_user_block_by_id(const char *username);

/**
 * Unblocks a user by the given id. The response memory needs to be freed by
 * the caller.
 */
gh_client_response_t*
gh_client_user_unblock_by_id(const char *username);

/**
 * Retrieve the list of followers for the logged in user. The response memory
 * needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_user_followers_list(const gh_client_req_list_opts_t *opts);

/**
 * Retrieve rate limit information for the authenticated user. The response
 * memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_user_rate_limit_info();

/**
 * List issues for the logged in user. The response memory needs to be freed by
 * the caller.
 */
gh_client_response_t*
gh_client_issues_for_user_list(const gh_client_issues_req_opts_t *opts);

/**
 * List issues for the given repository. The response memory needs to be freed
 * by the caller.
 */
gh_client_response_t*
gh_client_issues_by_repo_list(const char *owner, const char *repo,
                            const gh_client_issues_req_opts_t *opts);

/**
 * Create an issue. The response memory needs to be freed by the caller.
 *
 * data argument must be JSON in the following format:
 * 
 * title (required)
 * 
 * {"title":"Found a bug","body":"I'\''m having a problem with this.",
 *  "assignees":["octocat"],"milestone":1,"labels":["bug"]}
 */
gh_client_response_t*
gh_client_issue_create(const char *owner, const char *repo, const char *data);

/**
 * Retrieve the issue based on the given id. The response memory needs to be
 * freed by the caller.
 */
gh_client_response_t*
gh_client_issue_get(const char *owner, const char *repo,
                    const int unsigned id);

/**
 * Update the issue based on the given id. The response memory needs to be
 * freed by the caller.
 * 
 * data argument must be JSON in the following format:
 * 
 * {"title":"Found a bug","body":"I'\''m having a problem with this.",
 *  "assignees":["octocat"],"milestone":1,"state":"open","labels":["bug"]}
 */
gh_client_response_t*
gh_client_issue_update(const char *owner, const char *repo,
                    const int unsigned id, const char *data);

/**
 * Lock an issue. The response memory needs to be freed by the caller.
 * 
 * data argument must be JSON in the following format:
 * 
 * {"lock_reason":"off-topic"}
 * 
 * The API only returns a status code and not a body. A successful call will
 * have a code of 204. Please reference the API docs for an exhaustive list of
 * status codes.
 */
gh_client_response_t*
gh_client_issue_lock(const char *owner, const char *repo,
                    const int unsigned id, const char *data);

/**
 * Unlock an issue. The response memory needs to be freed by the caller.
 * 
 * The API only returns a status code and not a body. A successful call will
 * have a code of 204. Please reference the API docs for an exhaustive list of
 * status codes.
 */
gh_client_response_t*
gh_client_issue_unlock(const char *owner, const char *repo,
                    const int unsigned id);

/**
 * Retrieve the action billing information for the given organization. The
 * response memory needs to be freed by the caller.
 */
gh_client_response_t*
gh_client_actions_billing_by_org(const char *org);

/**
 * Retrieve all community metrics for the given repository.
 */
gh_client_response_t*
gh_client_metrics_community_profile(const char *owner, const char *repo);

/**
 * Retrieve total number of clones and breakdown per day or week for the last
 * 14 days. Valid values for interval are: "day", "week".
 */
gh_client_response_t*
gh_client_metrics_repository_clones(const char *owner, const char *repo,
                                    const char *interval);

/**
 * Retrieve the top 10 popular contents over the last 14 days.
 */
gh_client_response_t*
gh_client_metrics_top_referral_paths(const char *owner, const char *repo);

/**
 * Retrieve the top referrers over the last 14 days.
 */
gh_client_response_t*
gh_client_metrics_top_referral_sources(const char *owner, const char *repo);

/**
 * Retrieve total number of page views and breakdown per day or week for the
 * last 14 days. Valid values for interval are: "day", "week".
 */
gh_client_response_t*
gh_client_metrics_page_views(const char *owner, const char *repo,
                             const char *interval);

/**
 * Retrieve all codes of conduct. The response memory needs to be freed by the
 * caller.
 */
gh_client_response_t*
gh_client_codes_of_conduct_list();

/**
 * Retrieve a code of conduct by the given key. The response memory needs to be
 * freed by the caller.
 */
gh_client_response_t*
gh_client_code_of_conduct_get_by_key(const char *key);

/**
 * Free the memory used by the client.
 */
void
gh_client_free();

#endif /** end __CLIENT_H */
#ifdef __cplusplus
}
#endif
