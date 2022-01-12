module power

go 1.17

require (
	github.com/apibillme/cache v0.0.0-20180927200649-e0b3581c9b4d
	github.com/golang/mock v1.6.0
	github.com/joho/godotenv v1.4.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kryuchkovnet/protobuf v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/goleak v1.1.12
	google.golang.org/protobuf v1.27.1
)

replace github.com/kryuchkovnet/protobuf => ./protobuf/

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/smartystreets/goconvey v1.7.2 // indirect
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)
