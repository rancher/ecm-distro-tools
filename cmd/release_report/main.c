#include <inttypes.h>
#include <locale.h>
#include <pthread.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <wchar.h>

#include <jansson.h>

#include "github.h"

#define STR1(x) #x
#define STR(x) STR1(x)

#define RPM_TESTING_SUFFIX ".testing.0"
#define RPM_LATEST_SUFFIX ".latest.0"
#define RPM_STABLE_SUFFIX ".stable.0"

#define MAX_VERSIONS 10

struct release {
    char *org;
    char *repo;
    char *tag;
};

/**
 * org_from_repo receives a repository and returns the Github organization it 
 * belongs to.
 */
static inline const char*
org_from_repo(const char *repo)
{
    if (strcmp("rke2", repo) == 0 || strcmp("ecm-distro-tools", repo) == 0) {
        return "rancher";
    }

    if (strcmp("k3s", repo) == 0) {
        return "k3s-io";
    }

    return NULL;
}

/**
 * repo_from_tag receives a tag and returns the Github repository it is 
 * associated with.
 */
static inline const char*
repo_from_tag(const char *tag)
{
    if (strstr(tag, "rke2r") != NULL) {
        return "rke2";
    }
    if (strstr(tag, "k3s") != NULL) {
        return "k3s";
    }

    return NULL;
}

/**
 * rke2_rpm_release_info
 */
uint8_t
rke2_rpm_release_info(const struct release *rel)
{
    char full_tag[64] = {0};
    char *rpm_tags[3] = {
        RPM_TESTING_SUFFIX, RPM_LATEST_SUFFIX, RPM_STABLE_SUFFIX 
    };

    for (uint8_t i = 0; i < 3; i++) {
        if (full_tag[0] != '\0') {
            memset(full_tag, 0, 64);
        }

        strncpy(full_tag, rel->tag, 64);
        strncat(full_tag, rpm_tags[i], 64);

        gh_client_response_t *res = gh_client_repo_release_by_tag(rel->org, "rke2-packaging", full_tag);
        if (res->err_msg != NULL) {
            fprintf(stderr, "%s\n", res->err_msg);
            gh_client_response_free(res);
            return 1;
        }

        json_error_t error;
        json_t *json = json_loads(res->resp, 0, &error);
        if (json == NULL) {
            fprintf(stderr, "error: parsing JSON: %s\n", error.text);
            return 1;
        }

        const char *release_branch;
        const int prerelease;
        json_t *assets;
        int ret = json_unpack(json, "{s:s, s:b, s:o}",
                                    "target_commitish", &release_branch,
                                    "prerelease", &prerelease,
                                    "assets", &assets);
        if (ret) {
            fprintf(stderr, "error unpacking JSON\n");
            return 1;
        }
        size_t asset_count = json_array_size(assets);

        printf("RPMs %10s: %lu\n", rpm_tags[i], asset_count);

        json_decref(json);
        gh_client_response_free(res);
    }

    return 0;
}

/**
 * base_release_info retrieves and prints formatted 
 */
void*
base_release_info(void *arg)
{
    struct release *rel = (struct release*)arg;

    gh_client_response_t *res = gh_client_repo_release_by_tag(rel->org, rel->repo, rel->tag);
    if (res->err_msg != NULL) {
        fprintf(stderr, "%s\n", res->err_msg);
        gh_client_response_free(res);
        return (void*)1;
    }

    json_error_t error;
    json_t *json = json_loads(res->resp, 0, &error);
    if (json == NULL) {
        fprintf(stderr, "error: parsing JSON: %s\n", error.text);
        return (void*)1;
    }

    const char *branch;
    const int prerelease;
    json_t *assets;
    int ret = json_unpack(json, "{s:s, s:b, s:o}",
                                "target_commitish", &branch,
                                "prerelease", &prerelease,
                                "assets", &assets);
    if (ret) {
        fprintf(stderr, "error unpacking JSON\n");
        return (void*)1;
    }
    size_t asset_count = json_array_size(assets);

    printf("Tag:             %s\n", rel->tag);
    printf("Branch:          %s\n", branch);
    printf("Pre-Release:     %s\n", (prerelease ? "true" : "false"));
    printf("Assets:          %lu\n", asset_count);

    json_decref(json);
    gh_client_response_free(res);

    if (strcmp(rel->repo, "rke2") == 0) {
        uint8_t ret = rke2_rpm_release_info(rel);
        if (ret != 0) {
            return (void*)ret;
        }
    }

    printf("\n");

    return 0;
}

const struct release*
release_new(const char *org, const char *repo)
{
    struct release *rel = calloc(1, sizeof(struct release));
    rel->org = calloc(strlen(org)+1, sizeof(char));
    strcpy(rel->org, org);
    rel->repo = calloc(strlen(repo)+1, sizeof(char));
    strcpy(rel->repo, repo);

    return rel;
}

int
main(int argc, const char **argv)
{
    if (argc < 2) {
        fprintf(stderr, "error: tag required\n");
        return 1;
    }

    char *token = getenv("GITHUB_TOKEN");
    if (token == NULL || token[0] == '\0') {
        fprintf(stderr, "error: github token not set in environment or invalid\n");
        return 1;
    }

    if (gh_client_init(token)) {
        fprintf(stderr, "error: failed to initialize Github library\n");
        return 1;
    }

    setlocale(LC_CTYPE, "");

    char *repo = repo_from_tag(argv[1]);
    char *org = org_from_repo(repo);

    struct release *rel = release_new(org, repo);

    char *versions[MAX_VERSIONS];

    char *tkn = strtok(argv[1], ",");
    uint8_t i = 0;


    while (tkn != NULL && i < MAX_VERSIONS) {
        versions[i] = tkn;
        i++;
        tkn = strtok(NULL, ",");
    }

    pthread_t threads[MAX_VERSIONS];
    int thread_ids[MAX_VERSIONS];

    for (int j = 0; j < i; j++) {
        thread_ids[j] = j;

        rel->tag = calloc(strlen(versions[j])+1, sizeof(char));
        strcpy(rel->tag, versions[j]);

        int ret = pthread_create(&threads[j], NULL, base_release_info, (void*)rel);
        if (ret) {
            fprintf(stderr, "error: creating thread base release thread\n");
            return 1;
        }
        pthread_join(threads[j], NULL);
    }

    gh_client_free();

    wchar_t star = 0x2713;
    wprintf(L"\n%lc\n", star);

    return 0;
}

// Tag:            v1.33.1+rke2r1
// Branch:         release-1.33
// Pre_release:    false
// Assets:         74
// RPMs (testing): 60
// RPMs (latest):  60
// RPMs (stable):  60
