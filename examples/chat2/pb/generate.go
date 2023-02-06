package pb

//go:generate protoc -I. --go_opt=paths=source_relative --go_opt=Mchat2.proto=./pb  --go_out=. ./chat2.proto
