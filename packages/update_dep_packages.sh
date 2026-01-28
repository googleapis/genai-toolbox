cd server-darwin-arm64
npm install --force
rm -rf bin/ 
git add version.txt package.json package-lock.json

cd ../server-darwin-x64
npm install --force
rm -rf bin/ 
git add version.txt package.json package-lock.json


cd ../server-linux-x64
npm install --force
rm -rf bin/ 
git add version.txt package.json package-lock.json


cd ../server-win32-x64
npm install --force
rm -rf bin/ 
git add version.txt package.json package-lock.json

cd ..