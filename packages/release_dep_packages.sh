cd server-darwin-arm64
npm pack .
npm publish --access public

cd ../server-darwin-x64
npm pack .
npm publish --access public

cd ../server-linux-x64
npm pack .
npm publish --access public

cd ../server-win32-x64
npm pack .
npm publish --access public

cd ..
