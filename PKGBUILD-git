# Maintainer: cebem1nt <cebem1nt@gmail.com>

pkgname=dock-mango-git
pkgver=0.0.r0.g0000000
pkgrel=1
pkgdesc="GTK3-based dock for mangowm"
url="https://github.com/cebem1nt/dock-mango"
arch=('x86_64')
license=('MIT')
depends=('gtk3' 'gtk-layer-shell')
makedepends=('go' 'gobject-introspection')
optdepends=('nwg-drawer: default application launcher')

source=("git+${url}.git")
sha256sums=('SKIP')

pkgver() {
    cd "$srcdir/$pkgname"
    printf "r%s.%s" "$(git rev-list --count HEAD)" "$(git rev-parse --short HEAD)"
}

prepare(){
    cd "$srcdir/$pkgname"
    mkdir -p build/
}

build() {
    cd "$srcdir/$pkgname"
    export CGO_CPPFLAGS="${CPPFLAGS}"
    export CGO_CFLAGS="${CFLAGS}"
    export CGO_CXXFLAGS="${CXXFLAGS}"
    export CGO_LDFLAGS="${LDFLAGS}"
    export GOFLAGS="-buildmode=pie -trimpath -mod=readonly -modcacherw"
    go build -ldflags="-s -w -linkmode=external -buildid=''" -o "build/${pkgname}" *.go
}

package() {
    cd "$srcdir/$pkgname"
    install -Dm 755 "build/${pkgname}" "${pkgdir}/usr/bin/${pkgname}"
    install -Dm 644 config/* -t "${pkgdir}/usr/share/${pkgname}/"
    install -Dm 644 images/* -t "${pkgdir}/usr/share/${pkgname}/images/"
    install -Dm 644 LICENSE "${pkgdir}/usr/share/licenses/${pkgname}/LICENSE"
    install -Dm 644 README.md "${pkgdir}/usr/share/doc/${pkgname}/README.md"
}

