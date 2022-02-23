#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#define STR1(x) #x
#define STR(x) STR1(x)

// USAGE contains the application usage.
#define USAGE \
    "usage: %s [-vh] [-f]\n\
    -v          version\n\
    -h          help\n\
    -f          create a new file. default name: yyyy-mm-dd\n"

// STANDUP_TEMPLATE contains the base information to be filled out with 
// today's and tomorrow's dates.  This is then ready to be used to be 
// further fleshed out.
#define STANDUP_TEMPLATE \
"Yesterday:\n\
* \n\n\
Today:\n\
* \n\n\
PRs:\n\
* \n\n"

#define DATE_BUF_SZ 11

// DATE_FORMAT for getting the year, month, and day in yyyy-mm-dd format.
static const char* date_format = "%lu-%02lu-%02lu";

int
main(int argc, char **argv)
{
    int file_output = 0;
    FILE *out;

    int c;
    if (argc > 1) {
        while ((c = getopt(argc, argv, "hvf")) != -1) {
            switch (c) {
                case 'h':
                    printf(USAGE, STR(bin_name));
                    return 0;
                case 'v':
                    printf("%s %s - git: %s\n", 
                        STR(bin_name), 
                        STR(standup_version), 
                        STR(git_sha));
                    return 0;
                case 'f':
                    file_output = 1;
                    break;
                default:
                    printf(USAGE, STR(bin_name));
                    return 1;
            }
        }
    }

    time_t s = time(NULL);
    struct tm *now = localtime(&s);

    int year = now->tm_year + 1900;
    int month = now->tm_mon + 1;

    char today[DATE_BUF_SZ];
    sprintf(today, date_format, year, month, now->tm_mday);

    // check if the output needs to be written to a file. If so, set the output
    // to the file descriptor derived from "today's" date, otherwise set it to 
    // STDOUT.
    if (file_output) {
        out = fopen(today, "w");
    } else {
        out = stdout;
    }

    fprintf(out, STANDUP_TEMPLATE);
    fclose(out);

    return 0;
}
