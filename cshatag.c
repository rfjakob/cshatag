/**
 * Copyright (C) 2012 Jakob Unterwurzacher
 * Author(s): Jakob Unterwurzacher <jakobunt@gmail.com>
 *
 * This program is free software; you can redistribute it and/or
 * modify it under the terms of the GNU General Public License as
 * published by the Free Software Foundation; either version 2 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful, but
 * WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
 * General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program; if not, write to the Free Software
 * Foundation, Inc., 59 Temple Place - Suite 330, Boston, MA
 * 02111-1307, USA.
 */

#include <errno.h>
#include <openssl/sha.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <sys/xattr.h>
#include <time.h>

#define BUFSZ 8192
#define SHA256_BYTES 32
#define SHA256_NIBBLES 64

#ifdef __APPLE__
#define st_mtim st_mtimespec
#define fgetxattr(fd, name, buf, size) fgetxattr(fd, name, buf, size, 0, 0)
/*
 * SMB or MacOS bug: when working on an SMB mounted filesystem on a Mac, it seems the call
 * to `fsetxattr` does not update the xattr but removes it instead. So it takes two runs
 * of `cshatag` to update the attribute.
 * To work around this issue, we call `fsetxattr` twice ourselves.
 */
#define fsetxattr(fd, name, buf, size, flags) fsetxattr(fd, name, buf, size, 0, flags) \
                                              | fsetxattr(fd, name, buf, size, 0, flags)
#undef ENODATA
#define ENODATA ENOATTR
#endif

/**
 * Holds a file's metadata
 */
typedef struct {
    unsigned long long s;
    unsigned long ns;
    char sha256[SHA256_NIBBLES + 1]; // +1 for NULL byte
} xa_t;

/**
 * ASCII hex representation of char array
 */
char* bin2hex(unsigned char* bin, long len, char* out)
{
    char hexval[] = { '0', '1', '2', '3', '4', '5', '6', '7',
        '8', '9', 'a', 'b', 'c', 'd', 'e', 'f' };
    int j;
    for (j = 0; j < len; j++) {
        out[2 * j] = hexval[((bin[j] >> 4) & 0x0F)];
        out[2 * j + 1] = hexval[(bin[j]) & 0x0F];
    }
    out[2 * len] = 0;
    return out;
}

/**
 * sha256 of contents of f, ASCII hex representation
 */
char* fhash(FILE* f, char* hex)
{
    SHA256_CTX c;
    char buf[BUFSZ];
    size_t len;

    SHA256_Init(&c);

    while ((len = fread(buf, 1, BUFSZ, f))) {
        SHA256_Update(&c, buf, len);
    }

    unsigned char* hash;
    hash = calloc(1, SHA256_BYTES);
    if (NULL == hash) {
        fprintf(stderr, "Insufficient memory for hashing file");
        exit(EXIT_FAILURE);
    }
    SHA256_Final(hash, &c);

    return bin2hex(hash, SHA256_BYTES, hex);
}

/**
 * Nanosecond precision mtime of a file
 */
xa_t getmtime(FILE* f)
{
    xa_t actual;

    int fd = fileno(f);
    struct stat buf;
    fstat(fd, &buf);
    if (!S_ISREG(buf.st_mode)) {
        fprintf(stderr, "Error: this is not a regular file\n");
        exit(3);
    }
    actual.s = buf.st_mtim.tv_sec;
    actual.ns = buf.st_mtim.tv_nsec;

    return actual;
}

/**
 * File's actual metadata
 */
xa_t getactualxa(FILE* f)
{
    xa_t actual;
    /*
     * Must read mtime *before* file hash,
     * if the file is being modified, hash will be invalid
     * but timestamp will be outdated anyway
     */
    actual = getmtime(f);

    /*
     * Compute hash
     */
    fhash(f, actual.sha256);

    return actual;
}

/**
 * File's stored metadata
 */
xa_t getstoredxa(FILE* f)
{
    int fd = fileno(f);
    /* This initializes the whole struct, including the sha256[] array, to zero */
    xa_t xa = { 0 };

    /*
     * If fgetxattr fails, xa.sha256 stays all-zero.
     * "sizeof - 1" so we always have at least one null terminator.
     */
    int res = fgetxattr(fd, "user.shatag.sha256", xa.sha256, sizeof(xa.sha256));
    /* Make sure we have a NULL terminator */
    xa.sha256[sizeof(xa.sha256) - 1] = 0;
    if (res < 0 && errno != ENODATA) {
        perror("fgetxattr user.shatag.sha256 failed");
    }

    /*
     * Example:
     * 1335974989.123456789
     *    10     .     9     => len=20
     */

    /*
     * Initialize to all-zero so that:
     * 1) If fgetxattr fails we get a zero-length string
     * 2) If fgetxattr suceeds we have at least one null terminator
     */
    char ts[100] = { 0 };
    res = fgetxattr(fd, "user.shatag.ts", ts, sizeof(ts) - 1);
    if (res < 0 && errno != ENODATA) {
        perror("fgetxattr user.shatag.ts failed");
    }
    /*
     * If sscanf fails (because ts is zero-length) variables stay zero
     */
    sscanf(ts, "%llu.%lu", &xa.s, &xa.ns);

    return xa;
}

/**
 * Write out metadata to file's extended attributes
 */
int writexa(FILE* f, xa_t xa)
{
    int fd = fileno(f);
    int flags = 0, err = 0;

    char buf[100];
    snprintf(buf, sizeof(buf), "%llu.%09lu", xa.s, xa.ns);
    err = fsetxattr(fd, "user.shatag.ts", buf, strlen(buf), flags);
    err |= fsetxattr(fd, "user.shatag.sha256", &xa.sha256, SHA256_NIBBLES, flags);

    return err;
}

/**
 * Pretty-print metadata
 */
char* formatxa(xa_t s)
{
    char* buf;
    char* prettysha;
    int buflen = SHA256_NIBBLES + 100;
    buf = calloc(1, buflen);

    if (NULL == buf) {
        fprintf(stderr, "Insufficient space to store hash stringed");
        exit(EXIT_FAILURE);
    }

    if (strlen(s.sha256) > 0)
        prettysha = s.sha256;
    else
        prettysha = "0000000000000000000000000000000000000000000000000000000000000000";
    snprintf(buf, buflen, "%s %010llu.%09lu", prettysha, s.s, s.ns);
    return buf;
}

int main(int argc, const char* argv[])
{
    char const* myname;
    myname = argv[0];

    if (argc != 2) {
        fprintf(stderr, "Usage: %s FILE\n", myname);
        exit(1);
    }

    char const* fn;
    fn = argv[1];

    FILE* f;
    f = fopen(fn, "r");
    if (!f) {
        fprintf(stderr, "Error: could not open file \"%s\": %s\n", fn,
            strerror(errno));
        exit(2);
    }

    xa_t s;
    s = getstoredxa(f);
    xa_t a;
    a = getactualxa(f);
    int needsupdate = 0;
    int havecorrupt = 0;

    if (s.s == a.s && s.ns == a.ns) {
        /*
         * Times are the same, go ahead and compare the hash
         */
        if (strcmp(s.sha256, a.sha256) != 0) {
            /*
             * Hashes are different, but this may be because
             * the file has been modified while we were computing the hash.
             * So check if the mtime ist still the same.
             */
            xa_t a2;
            a2 = getmtime(f);
            if (s.s == a2.s && s.ns == a2.ns) {
                /*
                 * Now, this is either data corruption or somebody modified the file
                 * and reset the mtime to the last value (to hide the modification?)
                 */
                fprintf(stderr, "Error: corrupt file \"%s\"\n", fn);
                printf("<corrupt> %s\n", fn);
                printf(" stored: %s\n actual: %s\n", formatxa(s), formatxa(a));
                needsupdate = 1;
                havecorrupt = 1;
            }
        } else
            printf("<ok> %s\n", fn);
    } else {
        printf("<outdated> %s\n", fn);
        printf(" stored: %s\n actual: %s\n", formatxa(s), formatxa(a));
        needsupdate = 1;
    }

    if (needsupdate && writexa(f, a) != 0) {
        fprintf(stderr,
            "Error: could not write extended attributes to file \"%s\": %s\n",
            fn, strerror(errno));
        exit(4);
    }

    if (havecorrupt)
        return 5;
    else
        return 0;
}
