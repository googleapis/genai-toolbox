from aiohttp import ClientSession

from toolbox_llamaindex_sdk import ToolboxClient


def test_hello_world():
    print("Creating new Toolbox client.")
    session = ClientSession()
    client = ToolboxClient(url="test_url", session=session)
    print("Created toolbox client:", client)
    assert 0 == 0
