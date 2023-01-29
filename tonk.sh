#!/bin/sh

testdir="/dev/shm"
server="127.0.0.1:9999"

make clean
make
trap "cd ${testdir} && rm -rf honk views honk.db* blob.db*" 2 3
cp -a honk views data/views "${testdir}"
cd "${testdir}"
printf "mascal\nmascal\n${server}\n${server}\n" | ./honk init
printf "test\ntesttest\n" | ./honk adduser
./honk devel on
./honk
