#define _POSIX_C_SOURCE 200809L

#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <curl/curl.h>
#include <rattler.h>

#define STR1(x) #x
#define STR(x) STR1(x)

#define RANCHER_RAW_URL_FMT \
    "https://raw.githubusercontent.com/rancher/rancher/refs/heads/%s/build.yaml"
#define CHARTS_RAW_URL_FMT \
    "https://raw.githubusercontent.com/rancher/charts/refs/heads/%s/index.yaml"

#define HTTP_TIMEOUT_SECONDS 30L

#define MAX_BUILD_ENTRIES 64
#define MAX_CHART_RESULTS 16

typedef struct {
    char key[64];
    char version[128];
} build_entry;

// result of comparing one build.yaml key against the charts index.
typedef struct {
    const char *key;
    const char *chart;
    char version[128];
    int found; // 1 if chart/version exists in the index
} chart_status;

typedef struct {
    const char *key;
    const char *chart;
} key_chart_map_entry;

extern const key_chart_map_entry key_chart_map[];
extern const size_t key_chart_map_len;

const key_chart_map_entry key_chart_map[] = {
    { "webhookVersion", "rancher-webhook" },
    { "fleetVersion", "fleet" },
    { "turtlesVersion", "rancher-turtles" },
    { "provisioningCAPIVersion", "rancher-turtles" },
    { "cspAdapterMinVersion", "rancher-csp-adapter" },
    { "remoteDialerProxyVersion", "remotedialer-proxy" }
};
const size_t key_chart_map_len = sizeof(key_chart_map) / sizeof(key_chart_map[0]);

static int
chart_status_cmp(const void *a, const void *b)
{
    const chart_status *sa = a;
    const chart_status *sb = b;

    return strcmp(sa->key, sb->key);
}

size_t
build_results(const build_entry *build, size_t n_build, chart_status *results)
{
    size_t n_out = 0;
    for (size_t m = 0; m < key_chart_map_len; m++) {
        const char *key = key_chart_map[m].key;
        const build_entry *found_entry;

        found_entry = NULL;
        for (size_t b = 0; b < n_build; b++) {
            if (strcmp(build[b].key, key) == 0) {
                found_entry = &build[b];
                break;
            }
        }
        if (found_entry == NULL) {
            continue;
        }

        results[n_out].key = key_chart_map[m].key;
        results[n_out].chart = key_chart_map[m].chart;
        strncpy(results[n_out].version, found_entry->version,
            sizeof(results[n_out].version) - 1);
        results[n_out].version[sizeof(results[n_out].version) - 1] = '\0';
        results[n_out].found = 0;
        n_out++;
    }

    qsort(results, n_out, sizeof(*results), chart_status_cmp);
    return n_out;
}

size_t
count_missing(const chart_status *results, size_t n)
{
    size_t missing = 0;

    for (size_t i = 0; i < n; i++) {
        if (!results[i].found)
            missing++;
    }

    return missing;
}

void
print_report(FILE *out, const char *rancher_branch, const char *charts_branch,
             const chart_status *results, size_t n)
{
    size_t key_w = strlen("build.yaml");
    size_t chart_w = strlen("chart");
    size_t ver_w = strlen("version");

    for (size_t i = 0; i < n; i++) {
        size_t l = strlen(results[i].key);
        if (l > key_w) {
            key_w = l;
        }
        l = strlen(results[i].chart);
        if (l > chart_w) {
            chart_w = l;
        }
        l = strlen(results[i].version);
        if (l > ver_w) {
            ver_w = l;
        }
    }

    fprintf(out, "build.yaml: %s\n", rancher_branch);
    fprintf(out, "index.yaml: %s\n\n", charts_branch);

    fprintf(out, "%-*s  %-*s  %-*s  status\n",
        (int)key_w, "build.yaml",
        (int)chart_w, "chart",
        (int)ver_w, "version");
    fprintf(out, "%-*s  %-*s  %-*s  ------\n",
        (int)key_w, "----------",
        (int)chart_w, "-----",
        (int)ver_w, "-------");

    size_t missing = 0;
    for (size_t i = 0; i < n; i++) {
        const char *status = results[i].found ? "OK" : "MISSING";
        if (!results[i].found) {
            missing++;
        }

        fprintf(out, "%-*s  %-*s  %-*s  %s\n",
            (int)key_w, results[i].key,
            (int)chart_w, results[i].chart,
            (int)ver_w, results[i].version, status);
    }

    if (missing > 0) {
        fprintf(out, "\n%zu chart(s) missing from the charts index\n",
            missing);
    } else {
        fprintf(out, "\nall charts referenced in build.yaml are present "
            "in the charts index\n");
    }
}

/**
 * rstrip trims trailing \r, \n and whitespace in place.
 */
static void
rstrip(char *s)
{
    size_t n = strlen(s);

    while (n > 0 && isspace((unsigned char)s[n - 1])) {
        s[--n] = '\0';
    }
}

size_t
parse_build_yaml(char *text, build_entry *out, size_t max_out)
{
    char  *saveptr = NULL;
    size_t num_out = 0;

    for (char *line = strtok_r(text, "\n", &saveptr); line != NULL;
         line = strtok_r(NULL, "\n", &saveptr)) {

        rstrip(line);

        char *key_start = line;
        while (isspace((unsigned char)*key_start)) {
            key_start++;
        }

        // blank or comment line
        if (*key_start == '\0' || *key_start == '#') {
            continue;
        }

        char *colon = strchr(key_start, ':');
        if (colon == NULL) {
            continue; // not a "key: value" line
        }

        if (num_out >= max_out) {
            break;
        }

        *colon = '\0';
        char *val_start = colon + 1;
        while (isspace((unsigned char)*val_start)) {
            val_start++;
        }

        strncpy(out[num_out].key, key_start, sizeof(out[num_out].key) - 1);
        out[num_out].key[sizeof(out[num_out].key) - 1] = '\0';

        strncpy(out[num_out].version, val_start, sizeof(out[num_out].version) - 1);
        out[num_out].version[sizeof(out[num_out].version) - 1] = '\0';
        num_out++;
    }

    return num_out;
}

void
scan_index(char *text, chart_status *results, size_t n_results)
{
    char *saveptr = NULL;
    char current_chart[128] = "";

    for (char *line = strtok_r(text, "\n", &saveptr); line != NULL;
        line = strtok_r(NULL, "\n", &saveptr)) {
        size_t indent = 0;

        while (line[indent] == ' ') {
            indent++;
        }
        rstrip(line);
        size_t len = strlen(line);

        // top-level chart header, e.g. "  rancher-webhook:"
        if (indent == 2 && line[indent] != '-' && len > indent + 1 &&
            line[len - 1] == ':') {
            size_t namelen = len - indent - 1;

            if (namelen >= sizeof(current_chart)) {
                namelen = sizeof(current_chart) - 1;
            }
            memcpy(current_chart, line + indent, namelen);
            current_chart[namelen] = '\0';

            continue;
        }

        // chart version entry, e.g. "    version: 109.0.3+up0.10.7"
        if (indent == 4 && strncmp(line + indent, "version:", 8) == 0) {
            char *val = line + indent + 8;

            while (isspace((unsigned char)*val)) {
                val++;
            }

            for (size_t i = 0; i < n_results; i++) {
                if (strcmp(results[i].chart, current_chart) == 0 &&
                    strcmp(results[i].version, val) == 0) {
                    results[i].found = 1;
                }
            }
        }
    }
}

typedef struct {
    char *data;
    size_t len;
} body_buf;

void
body_buf_free(body_buf *buf)
{
    if (buf == NULL) {
        return;
    }

    free(buf->data);
    buf->data = NULL;
    buf->len = 0;
}

static size_t
write_cb(char *ptr, size_t size, size_t nmemb, void *userdata)
{
    body_buf *body = userdata;
    size_t add = size * nmemb;

    char *grown = realloc(body->data, body->len + add + 1);
    if (grown == NULL) {
        return 0;
    }

    body->data = grown;
    memcpy(body->data + body->len, ptr, add);
    body->len += add;
    body->data[body->len] = '\0';

    return add;
}

int
http_get(const char *url, body_buf *buf)
{
    buf->data = NULL;
    buf->len = 0;

    CURL *curl = curl_easy_init();
    if (curl == NULL) {
        fprintf(stderr, "error: curl_easy_init failed\n");
        return 1;
    }

    curl_easy_setopt(curl, CURLOPT_URL, url);
    curl_easy_setopt(curl, CURLOPT_WRITEFUNCTION, write_cb);
    curl_easy_setopt(curl, CURLOPT_WRITEDATA, buf);
    curl_easy_setopt(curl, CURLOPT_FOLLOWLOCATION, 1L);
    curl_easy_setopt(curl, CURLOPT_TIMEOUT, HTTP_TIMEOUT_SECONDS);
    curl_easy_setopt(curl, CURLOPT_USERAGENT, "charts-check/1.0");
    curl_easy_setopt(curl, CURLOPT_FAILONERROR, 0L);

    CURLcode res = curl_easy_perform(curl);
    if (res != CURLE_OK) {
        fprintf(stderr, "error: GET %s: %s\n", url, curl_easy_strerror(res));
        curl_easy_cleanup(curl);
        body_buf_free(buf);
        return 1;
    }

    long status_code;
    curl_easy_getinfo(curl, CURLINFO_RESPONSE_CODE, &status_code);
    curl_easy_cleanup(curl);

    if (status_code != 200) {
        fprintf(stderr, "error: GET %s: unexpected status %ld\n", url,
            status_code);
        body_buf_free(buf);
        return 1;
    }

    return 0;
}

/**
 * charts_branch_from_rancher_branch derives the rancher/charts branch name
 * from a rancher branch name, e.g. "release/v2.14" -> "release-v2.14". The
 * caller must free() the result.
 */
static char*
charts_branch_from_rancher_branch(const char *rancher_branch)
{
    char *out = strdup(rancher_branch);
    if (out == NULL) {
        return NULL;
    }

    char *slash = strchr(out, '/');
    if (slash != NULL) {
        *slash = '-';
    }

    return out;
}

static void
compare_cmd(rattler_cmd *cmd, int argc, char **argv)
{
    RATTLER_UNUSED(argc);

    const char *rancher_branch = argv[0];

    const char *charts_branch_flag = rattler_flag_string(cmd, "charts-branch");
    char *charts_branch = NULL;

    if (charts_branch_flag != NULL && *charts_branch_flag != '\0') {
        charts_branch = strdup(charts_branch_flag);
    } else {
        charts_branch = charts_branch_from_rancher_branch(rancher_branch);
    }
    if (charts_branch == NULL) {
        fprintf(stderr, "error: out of memory\n");
        return;
    }

    char rancher_url[256];
    snprintf(rancher_url, sizeof(rancher_url), RANCHER_RAW_URL_FMT,
        rancher_branch);

    char charts_url[256];
    snprintf(charts_url, sizeof(charts_url), CHARTS_RAW_URL_FMT,
        charts_branch);

    body_buf build_body = {0};
    if (http_get(rancher_url, &build_body) != 0) {
        free(charts_branch);
        return;
    }

    body_buf index_body = {0};
    if (http_get(charts_url, &index_body) != 0) {
        body_buf_free(&build_body);
        free(charts_branch);
        return;
    }

    build_entry build[MAX_BUILD_ENTRIES];
    size_t build_count = parse_build_yaml(build_body.data, build,
        MAX_BUILD_ENTRIES);

    chart_status results[MAX_CHART_RESULTS];
    size_t results_count = build_results(build, build_count, results);
    scan_index(index_body.data, results, results_count);

    print_report(stdout, rancher_branch, charts_branch,
        results, results_count);

    if (count_missing(results, results_count) > 0) {
        body_buf_free(&build_body);
        body_buf_free(&index_body);
        free(charts_branch);
        return;
    }

    body_buf_free(&build_body);
    body_buf_free(&index_body);
    free(charts_branch);
}

int
main(int argc, char **argv)
{
    curl_global_init(CURL_GLOBAL_DEFAULT);

    rattler_cmd *root = rattler_new_command(STR(bin_name) " [command]",
        "Rancher release helper CLI",
        "Central command for Rancher release verification tasks.");
    rattler_set_version(root, STR(charts_compare_version));

    rattler_cmd *compare = rattler_new_command(
        "compare [flags] <rancher-branch>",
        "Compare build.yaml chart versions against the charts index",
        "Fetches build.yaml from the given rancher release branch and\n"
        "index.yaml from the corresponding rancher/charts release\n"
        "branch, then reports any chart versions referenced in\n"
        "build.yaml that are missing from the charts index.\n\n"
        "The charts branch defaults to the rancher branch with '/'\n"
        "replaced by '-' (release/v2.14 -> release-v2.14); override\n"
        "with --charts-branch.");
    compare->run = compare_cmd;
    rattler_set_args(compare, 1, 1);
    rattler_flags_string(compare, "charts-branch", 'c', "",
        "override the derived rancher/charts branch");

    rattler_add_command(root, compare);

    int rc = rattler_execute(root, argc, argv);

    rattler_free(root);
    curl_global_cleanup();

    return rc;
}
