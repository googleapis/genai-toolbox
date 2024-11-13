from toolbox_llamaindex_sdk import ToolboxClient


def test_hello_world():
    print("Creating new Toolbox client.")
    client = ToolboxClient(url="test_url")
    print("Created toolbox client:", client)
    assert 0 == 0
