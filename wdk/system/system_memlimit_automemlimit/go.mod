module piko.sh/piko/wdk/system/system_memlimit_automemlimit

go 1.27.0

require (
	github.com/KimMachineGun/automemlimit v0.7.5
	go.uber.org/goleak v1.3.0
)

require (
	github.com/pbnjay/memory v0.0.0-20210728143218-7b4eea64cf58 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
)

replace piko.sh/piko => ../../../
