import yaml
import os

def safe_load_yaml(file_path):
    """
    安全加载 YAML 文件，处理常见的制表符缩进等问题。
    
    :param file_path: YAML 文件路径
    :return: 解析后的数据 (字典或列表)
    :raises: 如果加载失败，抛出 yaml.YAMLError 或 OSError
    """
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
            
        return yaml.safe_load(content)
    except yaml.scanner.ScannerError as e:
        # 检查错误是否可能是由制表符引起的
        if '\t' in content:
            # 尝试将制表符替换为 2 个空格 (常见约定)
            sanitized_content = content.replace('\t', '  ')
            try:
                return yaml.safe_load(sanitized_content)
            except yaml.YAMLError:
                # 如果 2 个空格不起作用，尝试 4 个空格
                sanitized_content = content.replace('\t', '    ')
                return yaml.safe_load(sanitized_content)
        raise e
    except Exception as e:
        raise e
