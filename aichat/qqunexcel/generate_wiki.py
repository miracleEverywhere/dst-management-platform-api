#!/usr/bin/env python3
"""将资料收集工作簿转换为 aichat Wiki Markdown 文档。"""

from __future__ import annotations

import argparse
import json
import os
import posixpath
import re
import tempfile
import zipfile
from dataclasses import dataclass, field
from pathlib import Path
from xml.etree import ElementTree


WORKBOOK_NAME = "饥荒管理平台AI功能资料收集.xlsx"
MANIFEST_NAME = ".wiki-generated.json"
MAIN_NS = "http://schemas.openxmlformats.org/spreadsheetml/2006/main"
REL_NS = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
PACKAGE_REL_NS = "http://schemas.openxmlformats.org/package/2006/relationships"
INVALID_FILENAME_RE = re.compile(r'[\\/:*?"<>|]')
CELL_REFERENCE_RE = re.compile(r"([A-Z]+)")
FIELD_BY_HEADER = {
    "描述": "description",
    "制作材料及解锁": "crafting",
    "如何获取": "acquisition",
    "备注": "notes",
}


@dataclass
class WikiDocument:
    name: str
    categories: list[str] = field(default_factory=list)
    description: list[str] = field(default_factory=list)
    crafting: list[str] = field(default_factory=list)
    acquisition: list[str] = field(default_factory=list)
    notes: list[str] = field(default_factory=list)
    source_rows: int = 0


def parse_args() -> argparse.Namespace:
    script_dir = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--workbook",
        type=Path,
        default=script_dir / WORKBOOK_NAME,
        help="输入的 xlsx 文件（默认：脚本同目录下的资料收集工作簿）",
    )
    parser.add_argument(
        "--output",
        type=Path,
        default=script_dir.parent / "wiki",
        help="Markdown 输出目录（默认：aichat/wiki）",
    )
    parser.add_argument(
        "--keep-stale",
        action="store_true",
        help="保留上次由本脚本生成、但本次工作簿中已不存在的文档",
    )
    return parser.parse_args()


def xml_root(archive: zipfile.ZipFile, name: str) -> ElementTree.Element:
    try:
        return ElementTree.fromstring(archive.read(name))
    except KeyError as error:
        raise ValueError(f"工作簿缺少必要文件：{name}") from error


def rich_text(element: ElementTree.Element | None) -> str:
    if element is None:
        return ""
    return "".join(node.text or "" for node in element.iter(f"{{{MAIN_NS}}}t"))


def read_shared_strings(archive: zipfile.ZipFile) -> list[str]:
    try:
        root = ElementTree.fromstring(archive.read("xl/sharedStrings.xml"))
    except KeyError:
        return []
    return [rich_text(item) for item in root.findall(f"{{{MAIN_NS}}}si")]


def column_index(reference: str) -> int:
    match = CELL_REFERENCE_RE.match(reference)
    if match is None:
        raise ValueError(f"无效的单元格引用：{reference}")
    result = 0
    for char in match.group(1):
        result = result * 26 + ord(char) - ord("A") + 1
    return result - 1


def cell_value(cell: ElementTree.Element, shared_strings: list[str]) -> str:
    cell_type = cell.get("t", "")
    if cell_type == "inlineStr":
        return rich_text(cell.find(f"{{{MAIN_NS}}}is"))

    value_node = cell.find(f"{{{MAIN_NS}}}v")
    value = value_node.text if value_node is not None and value_node.text is not None else ""
    if cell_type == "s" and value:
        index = int(value)
        if index >= len(shared_strings):
            raise ValueError(f"共享字符串索引越界：{index}")
        return shared_strings[index]
    if cell_type == "b":
        return "TRUE" if value == "1" else "FALSE"
    return value


def sheet_rows(
    archive: zipfile.ZipFile, sheet_path: str, shared_strings: list[str]
) -> list[list[str]]:
    root = xml_root(archive, sheet_path)
    rows: list[list[str]] = []
    for row_node in root.findall(f".//{{{MAIN_NS}}}sheetData/{{{MAIN_NS}}}row"):
        values: dict[int, str] = {}
        for cell in row_node.findall(f"{{{MAIN_NS}}}c"):
            index = column_index(cell.get("r", ""))
            values[index] = cell_value(cell, shared_strings)
        if values:
            row = [""] * (max(values) + 1)
            for index, value in values.items():
                row[index] = value
            rows.append(row)
        else:
            rows.append([])
    return rows


def read_workbook(path: Path) -> list[tuple[str, list[list[str]]]]:
    if not path.is_file():
        raise FileNotFoundError(f"找不到工作簿：{path}")

    with zipfile.ZipFile(path) as archive:
        workbook_root = xml_root(archive, "xl/workbook.xml")
        relationships_root = xml_root(archive, "xl/_rels/workbook.xml.rels")
        targets = {
            relationship.get("Id", ""): relationship.get("Target", "")
            for relationship in relationships_root.findall(
                f"{{{PACKAGE_REL_NS}}}Relationship"
            )
        }
        shared_strings = read_shared_strings(archive)
        sheets: list[tuple[str, list[list[str]]]] = []
        for sheet in workbook_root.findall(
            f".//{{{MAIN_NS}}}sheets/{{{MAIN_NS}}}sheet"
        ):
            name = sheet.get("name", "").strip()
            relationship_id = sheet.get(f"{{{REL_NS}}}id", "")
            target = targets.get(relationship_id)
            if not target:
                raise ValueError(f"找不到 Sheet {name} 对应的工作表文件")
            normalized_target = target.lstrip("/")
            if normalized_target.startswith("xl/"):
                sheet_path = posixpath.normpath(normalized_target)
            else:
                sheet_path = posixpath.normpath(posixpath.join("xl", normalized_target))
            sheets.append((name, sheet_rows(archive, sheet_path, shared_strings)))
        return sheets


def clean_cell(value: str) -> str:
    return value.replace("\r\n", "\n").replace("\r", "\n").strip()


def add_unique(values: list[str], value: str) -> None:
    if value and value not in values:
        values.append(value)


def safe_filename(name: str) -> str:
    filename = INVALID_FILENAME_RE.sub("_", name).rstrip(". ")
    if not filename:
        raise ValueError(f"条目名称无法生成有效文件名：{name!r}")
    return f"{filename}.md"


def collect_documents(
    sheets: list[tuple[str, list[list[str]]]],
) -> tuple[dict[str, WikiDocument], dict[str, int]]:
    documents: dict[str, WikiDocument] = {}
    category_rows: dict[str, int] = {}

    for category, rows in sheets[1:]:
        category_rows[category] = 0
        if not rows:
            continue
        headers = [clean_cell(value) for value in rows[0]]
        try:
            name_column = headers.index("名称")
        except ValueError:
            continue

        for row in rows[1:]:
            name = clean_cell(row[name_column]) if name_column < len(row) else ""
            if not name:
                continue
            category_rows[category] += 1
            document = documents.setdefault(name, WikiDocument(name=name))
            document.source_rows += 1
            add_unique(document.categories, category)

            for column, header in enumerate(headers):
                value = clean_cell(row[column]) if column < len(row) else ""
                if not value:
                    continue
                field_name = FIELD_BY_HEADER.get(header)
                if field_name is not None:
                    add_unique(getattr(document, field_name), value)
                elif header and header != "名称":
                    add_unique(document.notes, f"{header}：{value}")

    return documents, category_rows


def section(values: list[str]) -> str:
    return "\n".join(values) if values else "无"


def render_document(document: WikiDocument) -> str:
    return "\n".join(
        [
            "# 分类",
            section(document.categories),
            "",
            "# 描述",
            section(document.description),
            "",
            "# 制作材料及解锁",
            section(document.crafting),
            "",
            "# 如何获取",
            section(document.acquisition),
            "",
            "# 备注",
            section(document.notes),
            "",
        ]
    )


def atomic_write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        "w", encoding="utf-8", newline="\n", dir=path.parent, delete=False
    ) as temporary_file:
        temporary_file.write(content)
        temporary_path = Path(temporary_file.name)
    os.replace(temporary_path, path)


def read_manifest(path: Path) -> set[str]:
    if not path.is_file():
        return set()
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (json.JSONDecodeError, OSError) as error:
        raise ValueError(f"无法读取生成清单 {path}：{error}") from error
    if not isinstance(data, list) or not all(isinstance(item, str) for item in data):
        raise ValueError(f"生成清单格式无效：{path}")
    return set(data)


def write_documents(
    documents: dict[str, WikiDocument], output_dir: Path, keep_stale: bool
) -> tuple[list[str], list[str]]:
    output_dir.mkdir(parents=True, exist_ok=True)
    filenames: dict[str, str] = {}
    for document in documents.values():
        filename = safe_filename(document.name)
        collision_key = filename.casefold()
        previous_name = filenames.get(collision_key)
        if previous_name is not None and previous_name != document.name:
            raise ValueError(f"文件名冲突：{previous_name} / {document.name}")
        filenames[collision_key] = document.name
        atomic_write(output_dir / filename, render_document(document))

    current_files = {safe_filename(document.name) for document in documents.values()}
    manifest_path = output_dir / MANIFEST_NAME
    previous_files = read_manifest(manifest_path)
    stale_files = previous_files - current_files
    removed_files: list[str] = []
    if not keep_stale:
        for filename in sorted(stale_files):
            if Path(filename).name != filename or Path(filename).suffix.lower() != ".md":
                raise ValueError(f"生成清单中包含不安全路径：{filename}")
            stale_path = output_dir / filename
            if stale_path.is_file():
                stale_path.unlink()
                removed_files.append(filename)

    manifest_files = current_files | previous_files if keep_stale else current_files
    manifest = json.dumps(sorted(manifest_files), ensure_ascii=False, indent=2) + "\n"
    atomic_write(manifest_path, manifest)
    return sorted(current_files), removed_files


def main() -> None:
    args = parse_args()
    sheets = read_workbook(args.workbook.resolve())
    if len(sheets) < 2:
        raise ValueError("工作簿除填写指南外没有分类 Sheet")

    documents, category_rows = collect_documents(sheets)
    generated_files, removed_files = write_documents(
        documents, args.output.resolve(), args.keep_stale
    )
    source_rows = sum(category_rows.values())
    merged_names = sorted(
        document.name for document in documents.values() if document.source_rows > 1
    )
    populated_categories = [
        category for category, row_count in category_rows.items() if row_count > 0
    ]

    print(f"已跳过首个 Sheet：{sheets[0][0]}")
    print(f"已读取 {len(category_rows)} 个分类，其中 {len(populated_categories)} 个包含数据")
    print(f"已将 {source_rows} 行数据生成 {len(generated_files)} 份 Markdown 文档")
    if merged_names:
        print(f"已合并同名条目：{'、'.join(merged_names)}")
    if removed_files:
        print(f"已删除过期文档：{'、'.join(removed_files)}")


if __name__ == "__main__":
    main()
