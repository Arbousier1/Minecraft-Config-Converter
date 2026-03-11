from abc import ABC, abstractmethod
import os
import yaml
from src.utils.yaml_loader import safe_load_yaml

# 自定义 YAML Dumper 以确保列表缩进正确
class IndentDumper(yaml.Dumper):
    def increase_indent(self, flow=False, indentless=False):
        return super(IndentDumper, self).increase_indent(flow, False)

class RecipeDumper(IndentDumper):
    pass

def _recipe_represent_str(dumper, data):
    return dumper.represent_scalar("tag:yaml.org,2002:str", data, style='"')

RecipeDumper.add_representer(str, _recipe_represent_str)

class BaseConverter(ABC):
    def __init__(self):
        self.config = {}
        self.namespace = "converted"

    @abstractmethod
    def convert(self, data, namespace=None):
        """
        将输入数据转换为目标格式。
        :param data: 输入数据 (通常是字典)
        :param namespace: 命名空间
        :return: 转换后的数据
        """
        pass

    @abstractmethod
    def save_config(self, output_dir):
        """
        保存转换后的配置到输出目录。
        :param output_dir: 输出目录路径
        """
        pass

    def load_config(self, file_path):
        """
        加载 YAML 配置文件。
        :param file_path: 文件路径
        :return: 加载的数据
        """
        return safe_load_yaml(file_path)

    def _write_yaml_with_footer(self, data, file_path, dumper=None):
        """
        写入带有页脚注释的 YAML 文件。
        :param data: 要写入的数据
        :param file_path: 文件路径
        """
        os.makedirs(os.path.dirname(file_path), exist_ok=True)
        with open(file_path, 'w', encoding='utf-8') as f:
            if dumper is None:
                dumper = IndentDumper
            yaml.dump(data, f, Dumper=dumper, sort_keys=False, allow_unicode=True, default_flow_style=False)
            f.write("\n#该配置由 MCC Tool 自动生成 \n")
            f.write("#MCC Tool由闲鱼店铺：快乐售货铺 提供\n")
