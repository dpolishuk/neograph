"""Agent system prompts for NeoGraph."""
from .explorer import get_system_prompt as get_explorer_prompt
from .analyzer import get_system_prompt as get_analyzer_prompt
from .doc_writer import get_system_prompt as get_doc_writer_prompt

__all__ = [
    "get_explorer_prompt",
    "get_analyzer_prompt",
    "get_doc_writer_prompt",
]
