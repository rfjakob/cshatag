#!/usr/bin/env bats

function setup() {
	cd "$(dirname "$BATS_TEST_FILENAME")"
}

# Print hex encoded user.shatag.sha256 for file $1
function get_sha256() {
	if command -v getfattr > /dev/null 2>&1; then
		getfattr -n user.shatag.sha256 "$1" -e hex
	elif command -v xattr > /dev/null 2>&1; then
		xattr -x -p user.shatag.sha256 "$1"
	else
		exit 1
	fi
}

# Set extended attribute user.shatag.sha256=$1 (hex string) for the file $2
function set_sha256() {
	if command -v setfattr > /dev/null 2>&1; then
		setfattr -n user.shatag.sha256 -v "0x$1" "$2"
	elif command -v xattr > /dev/null 2>&1; then
		xattr -x -w user.shatag.sha256 "$1" "$2"
	else
		exit 1
	fi
}

# Set extended attribute user.shatag.ts=$1 (ascii string) for the file $2
function set_ts() {
	if command -v setfattr > /dev/null 2>&1; then
		setfattr -n user.shatag.ts -v "$1" "$2"
	elif command -v xattr > /dev/null 2>&1; then
		xattr -w user.shatag.ts "$1" "$2"
	else
		exit 1
	fi

}

@test "Testing new empty file" {
rm -f foo.txt
TZ=CET touch -t 201901010000 foo.txt
../cshatag foo.txt > 1.out
diff -u 1.expected 1.out
../cshatag foo.txt > 2.out
diff -u 2.expected 2.out
}

@test "Testing outdated file" {
echo changed > foo.txt
TZ=CET touch -t 202001010000 foo.txt
../cshatag foo.txt > 3.out
diff -u 3.expected 3.out
}

@test "Looking for NULL bytes (shouldn't find any)" {
if get_sha256 foo.txt | grep 00 ; then
	echo "error: NULL byte found"
	exit 1
fi
}

@test "Garbage on stderr?" {
rm -f foo.txt
echo > foo.txt
OUT=$(../cshatag foo.txt 2>&1 > /dev/null)
if [[ -n $OUT ]]; then
	echo "error: garbage on stderr: $OUT"
	exit 1
fi
}

@test "Testing modified empty file" {
echo > foo.txt
../cshatag foo.txt > /dev/null
../cshatag foo.txt > /dev/null
}

@test "Testing new 100-byte file" {
rm -f foo.txt
echo "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000" > foo.txt
../cshatag foo.txt > /dev/null
../cshatag foo.txt > /dev/null
}

@test "Testing cshatag v1.0 format with appended NULL byte" {
rm -f foo.txt
TZ=CET touch -t 201901010000 foo.txt
set_ts "1546297200.000000000" "foo.txt"
set_sha256 "6533623063343432393866633163313439616662663463383939366662393234323761653431653436343962393334636134393539393162373835326238353500" "foo.txt"
../cshatag foo.txt > /dev/null
}

@test "Testing shatag / cshatag v1.1 format without NULL byte" {
set_sha256 "65336230633434323938666331633134396166626634633839393666623932343237616534316534363439623933346361343935393931623738353262383535" "foo.txt"
../cshatag foo.txt > /dev/null
}

@test "Corrupt file should be flagged" {
echo "123" > foo.txt
TZ=CET touch -t 201901010000 foo.txt
run ../cshatag foo.txt &> /dev/null
[ "$status" -ne 0 ]
}

@test "Corrupt file should look ok on 2nd run" {
../cshatag foo.txt &> /dev/null
}

@test "Testing removal of extended attributes" {
rm -f foo.txt
TZ=CET touch -t 201901010000 foo.txt
../cshatag foo.txt > 1.out
diff -u 1.expected 1.out
../cshatag --remove foo.txt > 4.out
diff -u 4.expected 4.out
run bash -c "../cshatag --remove foo.txt 2> 5.err"

# MacOS returns ENOATTR instead of ENODATA on the remove
if [[ $(uname) == Darwin ]]
then
	diff -u 5.expected.mac 5.err
else
	diff -u 5.expected 5.err
fi
}

@test "Testing nonexisting file" {
run ../cshatag nonexisting.txt &> /dev/null
[ "$status" -eq 2 ]
}

@test "Testing symlink" {
ln -s / symlink1
run ../cshatag symlink1 &> /dev/null
[ "$status" -eq 3 ]
rm -f symlink1
}

@test "Testing timechange" {
echo same > foo.txt
TZ=CET touch -t 201901010000 foo.txt
../cshatag foo.txt > /dev/null
TZ=CET touch -t 201901010001 foo.txt
../cshatag foo.txt > 6.out
diff -u 6.expected 6.out
rm foo.txt
}

@test "Testing recursive flag" {
rm -rf foo
mkdir foo
TZ=CET touch -t 201901010000 foo/foo.txt
run bash -c "../cshatag foo 2> 7.err"
[ "$status" -eq 3 ]
diff -u 7.expected 7.err
../cshatag --recursive foo > 8.out
diff -u 8.expected 8.out
rm -rf foo
}

@test 'Testing -dry-run' {
TZ=CET touch -t 201901010000 foo.txt
../cshatag -dry-run foo.txt > 9.out
diff -u 9.expected 9.out
# Because with -n we have made no changes, we get the same output again.
../cshatag foo.txt > 9.out2
diff -u 9.expected 9.out2
}

@test 'Testing -dry-run -remove' {
../cshatag -dry-run -remove foo.txt > 11.out
diff -u 11.expected 11.out
# Because with -n we have made no changes, we get the same output again.
../cshatag -remove foo.txt > 11.out2
diff -u 11.expected 11.out2
}

@test 'Testing 100ns resolution' {
# https://github.com/rfjakob/cshatag/issues/21
rm -rf foo.txt
# Datestring generated using "date --rfc-3339=ns"
touch --date="2023-04-16 20:56:16.585798497+02:00" foo.txt
../cshatag foo.txt > /dev/null
echo asd > foo.txt
touch --date="2023-04-16 20:56:16.585798400+02:00" foo.txt
run ../cshatag foo.txt &> /dev/null
[ "$status" -eq 5 ]
}
