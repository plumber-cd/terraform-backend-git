%global commit      f9518cf8ed9e56bee0f0ed97dd708c06b56c2900
%global shortcommit %(c=%{commit}; echo ${c:0:7})

Name:           terraform-backend-git
Version:        0.0.18
Release:        1%{?dist}
Summary:        This application is an example for the golang binary RPM spec
License:        MIT
URL:            https://github.com/rigrassm/terraform-backend-git
Source0:	https://github.com/rigrassm/terraform-backend-git/archive/v%{version}.tar.gz

#BuildRequires:  golang >= 1.13
# pull in golang libraries by explicit import path, inside the meta golang()
#BuildRequires:  golang(github.com/gorilla/mux) >= 0-0.13
#BuildRequires:  golang(github.com/go-git/go-billyv) >= 5.0.0
#BuildRequires:  golang(github.com/go-git/go-git) >= 5.0.0
#BuildRequires:  golang(github.com/gorilla/handlers) >= 1.4.2
#BuildRequires:  golang(github.com/hashicorp/terraform) >= 0.12.24
#BuildRequires:  golang(github.com/mitchellh/go-homedir) >= v1.1.0
#BuildRequires:  golang(github.com/spf13/cobra) >= 1.0.0
#BuildRequires:  golang(github.com/spf13/viper) >= 1.4.0
#BuildRequires:  golang(golang.org/x/crypto) >= 0.0.0-20200406173513-056763e48d71
#BuildRequires:  golang(golang.org/x/sys) >= 0.0.0-20200409092240-59c9f1ba88fa

%description
# include your full description of the application here.
Terraform State backend for Git

%prep
%autosetup -n %{name}-%{version}

# many golang binaries are "vendoring" (bundling) sources, so remove them. Those dependencies need to be packaged independently.
rm -rf vendor

%build
# set up temporary build gopath, and put our directory there
export GO111MODULE=on
mkdir -p ./_build/src/github.com/rigrassm/terraform-backend-git
ln -s $(pwd) ./_build/src/github.com/rigrassm/terraform-backend-git

export GOPATH=/root/go:$(pwd)/_build
go build -ldflags "-B 0x$(head -c20 /dev/urandom|od -An -tx1|tr -d ' \n')" -o terraform-backend-git .

%install
install -d %{buildroot}%{_bindir}
install -p -m 0755 ./terraform-backend-git %{buildroot}%{_bindir}/terraform-backend-git

%files
%doc CHANGELOG.md LICENSE README.md
%{_bindir}/terraform-backend-git

%changelog
* Tue Oct 20 2020 Jill User <jill.user@fedoraproject.org> - 0.0.17-1
- package the terraform-backend-git
