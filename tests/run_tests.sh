#!/bin/bash

set -eu

cd "$(dirname "$0")"

echo "*** Testing new empty file ***"
rm -f foo.txt
echo > foo.txt
../cshatag foo.txt
../cshatag foo.txt

echo "*** Testing modified empty file ***"
echo > foo.txt
../cshatag foo.txt
../cshatag foo.txt

echo "*** Testing new 100-byte file ***"
dd if=/dev/zero of=foo.txt bs=100 count=1
../cshatag foo.txt
../cshatag foo.txt

echo "*** Testing cshatag v1.0 format with appended NULL byte ***"
rm -f foo.txt
touch -t 201901010000 foo.txt
setfattr -n user.shatag.ts -v "1546297200.000000000" foo.txt
setfattr -n user.shatag.sha256 -v 0x6533623063343432393866633163313439616662663463383939366662393234323761653431653436343962393334636134393539393162373835326238353500 foo.txt
../cshatag foo.txt

echo "*** Testing shatag / cshatag v1.1 format without NULL byte ***"
setfattr -n user.shatag.sha256 -v 0x65336230633434323938666331633134396166626634633839393666623932343237616534316534363439623933346361343935393931623738353262383535 foo.txt
../cshatag foo.txt

echo "*** ALL TESTS PASSED ***"
