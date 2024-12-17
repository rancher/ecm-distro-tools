#define __USE_XOPEN
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#include <jansson.h>

#include "github.h"
#include "spinner.h"

#define RFC3339_JAN_1 "2024-01-01T00:00:00Z"
#define MAX_RELEASE_COUNT 1024

/**
 * str_to_time converts the given date/time string into a time_t value.
 */
static time_t
str_to_time(const char *str)
{
    if (str == NULL) {
        return -1;
    }

    struct tm tm_struct = {0};

    if (strptime(str, "%Y-%m-%dT%H:%M:%SZ", &tm_struct) == NULL) {
        return 1;
    }

    return mktime(&tm_struct);
}

static char const *releases[MAX_RELEASE_COUNT] = {0};

static unsigned int brian = 0;
static unsigned int brooks = 0;
static unsigned int nicholas = 0;
static unsigned int pedro = 0;
static unsigned int rafael = 0;

static void
update_author_counts(const char *author)
{
    if (author == NULL) {
        return;
    }

    if (strcmp("briandowns", author) == 0) brian++;
    if (strcmp("rafaelbreno", author) == 0) rafael++;
    if (strcmp("nicholasSUSE", author) == 0) nicholas++;
    if (strcmp("brooksn", author) == 0) brooks++;
    if (strcmp("tashima42", author) == 0) pedro++;
}

int
main(int argc, char **argv)
{
    if (argc != 3) {
        fprintf(stderr, "usage %s <org> <repo>\n", argv[0]);
        return 1;   
    }

    const char *org = argv[1];
    const char *repo = argv[2];

    char *token = getenv("GITHUB_TOKEN");
    if (token == NULL || token[0] == '\0') {
        fprintf(stderr, "github token not set in environment or invalid\n");
        return 1;
    }

    spinner_t *s = spinner_new(31);
    s->delay = 100000;
    spinner_start(s);

    gh_client_init(token);

    time_t jan_1_2024 = str_to_time(RFC3339_JAN_1);
    if (jan_1_2024 == -1) {
        fprintf(stderr, "failed to parse: " RFC3339_JAN_1 "\n");
        return 1;
    }

    gh_client_req_list_opts_t opts = {
        .per_page = 50
    };
    gh_client_response_t *res = gh_client_repo_releases_list(org, repo, &opts);
    if (res->err_msg != NULL) {
        fprintf(stderr, "%s\n", res->err_msg);
        gh_client_response_free(res);
        return 1;
    }

    json_error_t error;
    json_t *root = json_loads(res->resp, 0, &error);

    int next_block = 0;

    size_t index;
    json_t *value, *author_obj, *login;

    const char *tag_name = {0};
    const char *created_at = {0};
    const char *author_name = {0};
    
    json_array_foreach(root, index, value) {
        json_unpack(value, "{s:s, s:s, s:o}",
            "tag_name", &tag_name,
            "created_at", &created_at);

        time_t cai = str_to_time(created_at);
        if (jan_1_2024 == -1) {
            fprintf(stderr, "failed to parse: %s\n", created_at);
            return 1;
        }

        if (cai > jan_1_2024) {
            releases[next_block] = tag_name;
            next_block++;
            printf("XXX - %s\n", res->resp);
            author_obj = json_object_get(value, "author");
            if (!json_is_object(author_obj)) {
                fprintf(stderr, "error: 'author' not an aobject\n");
                json_decref(root);
                return 1;
            }
            
            author_obj = json_object_get(value, "author");
            json_unpack(author_obj, "{s:s}", "login", &author_name);

            update_author_counts(author_name);
        }
    }

    while (res->next_link != NULL) {
        gh_client_req_list_opts_t opts = {
            .page_url = res->next_link
        };
        res = gh_client_repo_releases_list(org, repo, &opts);
        if (res->err_msg != NULL) {
            fprintf(stderr, "%s\n", res->err_msg);
            gh_client_response_free(res);
            return 1;
        }

        root = json_loads(res->resp, 0, &error);

        json_array_foreach(root, index, value) {
            json_unpack(value, "{s:s, s:s, s:o}",
                "tag_name", &tag_name,
                "created_at", &created_at);

            time_t cai = str_to_time(created_at);
            if (jan_1_2024 == -1) {
                fprintf(stderr, "failed to parse: %s\n", created_at);
                return 1;
            }

            if (cai > jan_1_2024) {
                releases[next_block] = tag_name;
                next_block++;
                
                author_obj = json_object_get(value, "author");
                json_unpack(author_obj, "{s:s}", "login", &author_name);

                update_author_counts(author_name);
            }
        }

        res->resp = NULL;
    }

    int rel_count = 0;
    int rel_rc_count = 0;
    
    for (int i = 0; i < MAX_RELEASE_COUNT; i++) {
        if (releases[i] == NULL || *releases[0] == '\0') continue;
        
        if (strstr(releases[i], "-rc") != NULL) {
            rel_rc_count++;
        } else {
            rel_count++;
        }
    }

    spinner_stop(s);
    spinner_free(s);

    printf("+ ------------- | ---- +\n");
    printf("| Releases      | No.  |\n");
    printf("+ --------------+ ---- +\n");
    printf("| GA            | %4d |\n", rel_count);
    printf("| RCs           | %4d |\n", rel_rc_count);
    printf("+ --------------+ ---- +\n");
    printf("| Total         | %4d |\n", rel_count + rel_rc_count);
    printf("+ --------------+ ---- +\n\n");

    printf("+ ------------- | ---- +\n");
    printf("| Captain       | No.  |\n");
    printf("+ ------------- + ---- +\n");
    printf("| Brooks        | %4d |\n", brooks);
    printf("| Rafael        | %4d |\n", rafael);
    printf("| Brian         | %4d |\n", brian);
    printf("| Pedro         | %4d |\n", pedro);
    printf("| Nichaolas     | %4d |\n", nicholas);
    printf("+ ------------- + ---- +\n");

    json_decref(value);
    json_decref(author_obj);
    json_decref(login);
    json_decref(root);

    gh_client_response_free(res);
    gh_client_free();

    return 0;
}

//
// EOY Release Report
//

// RKE2
// ----------
// Full: 46
// RCs: 153
// Total: 199

// K3s
// ----------
// Full: 51
// RCs: 94
// Total: 145

// Rancher
// ----------
// Full: 142
// RCs: 85
// Total: 227
