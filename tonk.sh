#!/bin/sh

testdir="/dev/shm"
server="127.0.0.1:9999"

make clean
make
make test || exit 1
trap "cd ${testdir} && rm -rf honk views honk.db* blob.db*" 2 3
cp -a honk views "${testdir}"
cd "${testdir}"
printf "mascal\nmascal\n${server}\n${server}\n" | ./honk init
./honk
