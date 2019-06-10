#!/bin/bash

set -eu

# Check that we have getfattr / setfattr
getfattr --version > /dev/null
setfattr --version > /dev/null

cd "$(dirname "$0")"

echo "*** Testing new empty file ***"
rm -f foo.txt
touch -t 201901010000 foo.txt
../cshatag foo.txt > 1.out
diff -u 1.expected 1.out
../cshatag foo.txt > 2.out
diff -u 2.expected 2.out

echo "*** Looking for NULL bytes (shouldn't find any)***"
if getfattr -n user.shatag.sha256 foo.txt -e hex | grep 00 ; then
	echo "error: NULL byte found"
	exit 1
fi

echo "*** Garbage on stderr? ***"
rm -f foo.txt
echo > foo.txt
OUT=$(../cshatag foo.txt 2>&1 > /dev/null)
if [[ -n $OUT ]]; then
	echo "error: garbage on stderr: $OUT"
	exit 1
fi

echo "*** Testing modified empty file ***"
echo > foo.txt
../cshatag foo.txt > /dev/null
../cshatag foo.txt > /dev/null

echo "*** Testing new 100-byte file ***"
rm -f foo.txt
dd if=/dev/zero of=foo.txt bs=100 count=1 status=none
../cshatag foo.txt > /dev/null
../cshatag foo.txt > /dev/null

echo "*** Testing cshatag v1.0 format with appended NULL byte ***"
rm -f foo.txt
touch -t 201901010000 foo.txt
setfattr -n user.shatag.ts -v "1546297200.000000000" foo.txt
setfattr -n user.shatag.sha256 -v 0x6533623063343432393866633163313439616662663463383939366662393234323761653431653436343962393334636134393539393162373835326238353500 foo.txt
../cshatag foo.txt > /dev/null

echo "*** Testing shatag / cshatag v1.1 format without NULL byte ***"
setfattr -n user.shatag.sha256 -v 0x65336230633434323938666331633134396166626634633839393666623932343237616534316534363439623933346361343935393931623738353262383535 foo.txt
../cshatag foo.txt > /dev/null

echo "*** Corrupt file ***"
echo "123" > foo.txt
touch -t 201901010000 foo.txt
set +e
../cshatag foo.txt &> /dev/null
RES=$?
set -e
if [[ $RES -eq 0 ]]; then
	echo "should have returned an error code, but returned 0"
	exit 1
fi

echo "*** ALL TESTS PASSED ***"
