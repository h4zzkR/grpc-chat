package main

import "time"

const UidLength = 32
const ClientConnectionTimeout = 2 * time.Second
const ClientStreamCapacity = 1000

const RedisCacheAddr = "localhost:7777"
const RedisCacheDBName = RedisCacheAddr + "_history"
const RedisCachePass = ""