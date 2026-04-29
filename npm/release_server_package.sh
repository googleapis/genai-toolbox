cd server
npm install --package-lock-only
git add package.json package-lock.json
git commit -m "update dep versions"

npm pack .
npm publish --access public