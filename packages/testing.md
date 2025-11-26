# How to test on different platforms

1. Create a local test registry. We are using Verdaccio. During the actual process, we publish to npm instead.
    ```sh
    # install
    npm install -g verdaccio

    # start server
    verdaccio
    ```

    The output will show the config file location and the URL. Open your browser to:
    http://localhost:4873

    > Note: You can use `npm unpublish > server-darwin-arm64@1.0.0 --force --registry http://localhost:4873` to unpublish the package.

2. Pack all 4 packages and publish them to the local registry. Go in each package (eg. server-darwin-arm64) and run
    ```sh
    npm ci --force
    npm pack .
    ```
    
    Then publish to the local registry
    ```sh
    npm publish --registry http://localhost:4873
    ```

3. Go to the server package and run
    ```sh
    npm ci --force
    npm pack .
    npm publish --registry http://localhost:4873
    ```

    Now, you have published your package.

4. Now add a tools.yaml file to the server folder. It should look like this:
    ```yaml
      sources:
        my-pg-source:
          kind: postgres
          host: 127.0.0.1
          port: 5432
          database: toolbox_db
          user: postgres
          password: password
      tools:
        search-hotels-by-name:
          kind: postgres-sql
          source: my-pg-source
          description: Search for hotels based on name.
          parameters:
            - name: name
              type: string
              description: The name of the hotel.
          statement: SELECT * FROM hotels WHERE name ILIKE '%' || $1 || '%';
      prompts:
        code-review:
          description: "Asks the LLM to analyze code quality and suggest improvements."
          messages:
            - role: "user"
              content: "Please review the following code for quality, correctness, and potential improvements: \n\n{{.code}}"
          arguments:
            - name: "code"
              description: "The code to review"
              required: true
    ```

5. From the packages/server folder, run 
    ```sh
    npx --registry=http://localhost:4873/ -y @toolbox-sdk/server
    ```

    This should start up the toolbox server with the tools.yaml file.

6. Run the command to verify that the tools are available:

    ```sh
    curl --location 'http://127.0.0.1:5000/mcp/tools/list' \
    --header 'Content-Type: application/json' \
    --data '{
        "jsonrpc": "2.0",
        "method": "tools/list",
        "params": {},
        "id": 1
    }'

```
