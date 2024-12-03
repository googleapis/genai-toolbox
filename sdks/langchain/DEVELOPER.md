# Development

Below are the details to set up a development environment and run tests.

## Install
1. Clone the repository:
    ```bash
    git clone https://github.com/googleapis/genai-toolbox.git
    ```
1. Navigate to the SDK directory:
    ```bash
    cd genai-toolbox/sdks/langchain
    ```
1. Install the package in editable mode, so changes are reflected without reinstall:
    ```bash
    pip install -e .
    ```
1. Make code changes and contribute to the SDK's development.
> [!TIP]
> Using `-e` option allows you to make changes to the SDK code and have
> those changes reflected immediately without reinstalling the package.

## Test
1. Navigate to the SDK directory:
    ```bash
    cd genai-toolbox/sdks/langchain
    ```
1. Install the SDK and test dependencies:
    ```bash
    pip install -e .[test]
    ```
1. Run tests and/or contribute to the SDK's development.

    ```bash
    pytest
    ```