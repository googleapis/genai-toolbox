# Central registries for managing Source and Tool configurations and their instances.
from typing import Callable, Dict, Type, Any, Optional
from py_toolbox.internal.sources.base import SourceConfig, Source
from py_toolbox.internal.core.logging import get_logger
import yaml

logger = get_logger(__name__)

SourceConfigFactory = Callable[[str, Dict[str, Any]], SourceConfig]

class SourceRegistry:
    def __init__(self):
        self._factories: Dict[str, SourceConfigFactory] = {}

    def register(self, kind: str, factory: SourceConfigFactory):
        if kind in self._factories:
            logger.warning(f"Source kind '{kind}' already registered. Overwriting.")
        self._factories[kind] = factory
        logger.info(f"Source kind '{kind}' registered.")

    def get_source_config(self, name: str, config_data: Dict[str, Any]) -> SourceConfig:
        kind = config_data.get("kind")
        if not kind:
            raise ValueError(f"Configuration for source '{name}' must include a 'kind' field.")

        factory = self._factories.get(kind)
        if not factory:
            raise ValueError(f"Unknown source kind: '{kind}' for source '{name}'. Ensure it's registered.")

        try:
            # The factory is expected to call the appropriate SourceConfig subclass's from_dict or constructor
            return factory(name, config_data)
        except Exception as e:
            logger.error(f"Error creating source config for '{name}' (kind: {kind}): {e}")
            raise ValueError(f"Failed to create source config for '{name}': {e}") from e

    def load_sources_from_config(self, file_path: str) -> Dict[str, Source]:
        logger.info(f"Loading sources from configuration file: {file_path}")
        try:
            with open(file_path, 'r') as f:
                config = yaml.safe_load(f)
        except FileNotFoundError:
            logger.error(f"Configuration file not found: {file_path}")
            raise
        except yaml.YAMLError as e:
            logger.error(f"Error parsing YAML configuration file {file_path}: {e}")
            raise

        if not config or 'sources' not in config:
            logger.warning(f"No 'sources' section found in {file_path}.")
            return {}

        initialized_sources: Dict[str, Source] = {}
        for name, source_config_data in config['sources'].items():
            if not isinstance(source_config_data, dict):
                logger.error(f"Source configuration for '{name}' is not a dictionary. Skipping.")
                continue
            try:
                logger.debug(f"Processing source config for '{name}': {source_config_data}")
                source_cfg_instance = self.get_source_config(name, source_config_data)
                source_instance = source_cfg_instance.initialize()
                initialized_sources[name] = source_instance
                logger.info(f"Successfully initialized source '{name}' of kind '{source_instance.source_kind()}'.")
            except ValueError as e:
                logger.error(f"Failed to initialize source '{name}': {e}")
                # Decide if one failure should stop all, for now, it continues
            except Exception as e:
                logger.error(f"An unexpected error occurred while initializing source '{name}': {e}")

        return initialized_sources

# Placeholder for ToolRegistry if we decide to put it here.
# For now, focusing on SourceRegistry.

from py_toolbox.internal.tools.base import ToolConfig, Tool, Manifest, McpManifest

ToolConfigFactory = Callable[[str, Dict[str, Any]], ToolConfig]

class ToolRegistry:
    def __init__(self):
        self._factories: Dict[str, ToolConfigFactory] = {}

    def register(self, kind: str, factory: ToolConfigFactory):
        if kind in self._factories:
            logger.warning(f"Tool kind '{kind}' already registered. Overwriting.")
        self._factories[kind] = factory
        logger.info(f"Tool kind '{kind}' registered.")

    def get_tool_config(self, name: str, config_data: Dict[str, Any]) -> ToolConfig:
        kind = config_data.get("kind")
        if not kind:
            raise ValueError(f"Configuration for tool '{name}' must include a 'kind' field.")

        factory = self._factories.get(kind)
        if not factory:
            raise ValueError(f"Unknown tool kind: '{kind}' for tool '{name}'. Ensure it's registered.")

        try:
            # Pass the whole config_data to the factory, which then calls from_dict
            return factory(name, config_data)
        except Exception as e:
            logger.error(f"Error creating tool config for '{name}' (kind: {kind}): {e}")
            raise ValueError(f"Failed to create tool config for '{name}': {e}") from e

    def load_tools_from_config(self, file_path: str, initialized_sources: Mapping[str, Source]) -> Dict[str, Tool]:
        logger.info(f"Loading tools from configuration file: {file_path}")
        try:
            with open(file_path, 'r') as f:
                config = yaml.safe_load(f)
        except FileNotFoundError:
            logger.error(f"Configuration file not found: {file_path}")
            raise
        except yaml.YAMLError as e:
            logger.error(f"Error parsing YAML configuration file {file_path}: {e}")
            raise

        if not config or 'tools' not in config:
            logger.warning(f"No 'tools' section found in {file_path}.")
            return {}

        initialized_tools: Dict[str, Tool] = {}
        for name, tool_config_data in config['tools'].items():
            if not isinstance(tool_config_data, dict):
                logger.error(f"Tool configuration for '{name}' is not a dictionary. Skipping.")
                continue
            try:
                logger.debug(f"Processing tool config for '{name}': {tool_config_data}")
                tool_cfg_instance = self.get_tool_config(name, tool_config_data)
                # Pass the map of already initialized sources to the tool's initialize method
                tool_instance = tool_cfg_instance.initialize(initialized_sources)
                initialized_tools[name] = tool_instance
                logger.info(f"Successfully initialized tool '{name}' of kind '{tool_instance.tool_kind()}'.")
            except ValueError as e:
                logger.error(f"Failed to initialize tool '{name}': {e}")
            except Exception as e:
                logger.error(f"An unexpected error occurred while initializing tool '{name}': {e}")

        return initialized_tools
