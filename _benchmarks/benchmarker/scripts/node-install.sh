INSTALL_DIR=$1
VERSION=$2
cd $INSTALL_DIR
curl http://nodejs.org/dist/node-$VERSION.tar.gz | tar xz --strip-components=1
./configure --prefix=$INSTALL_DIR
make install
curl https://www.npmjs.org/install.sh | sh