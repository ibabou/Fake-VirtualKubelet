module fake-vn

go 1.12

replace k8s.io/api => k8s.io/api v0.0.0-20190606204050-af9c91bd2759

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d

replace k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190606204521-b8faab9c5193+incompatible

replace k8s.io/kubernetes => k8s.io/kubernetes v1.14.3

require (
	github.com/sirupsen/logrus v1.4.2
	github.com/virtual-kubelet/node-cli v0.1.2
	github.com/virtual-kubelet/virtual-kubelet v1.0.0
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/kubernetes v1.13.7
	k8s.io/utils v0.0.0-20190923111123-69764acb6e8e // indirect
)
