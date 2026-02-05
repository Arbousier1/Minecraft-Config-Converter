# build.py
import PyInstaller.__main__
import os
import shutil

# 清理以前的构建
if os.path.exists('build'):
    shutil.rmtree('build')
if os.path.exists('dist'):
    shutil.rmtree('dist')

# 定义构建参数
args = [
    'web/app.py',                      # 主脚本
    '--name=MCC_Tool',                 # 可执行文件名称
    '--onefile',                       # 创建单个可执行文件
    '--icon=icon.png',                 # 使用自定义图标
    # '--noconsole',                   # 建议保留控制台以便查看服务器日志和关闭程序
    '--add-data=web/templates;templates', # 包含 HTML 模板
    '--add-data=web/static;static',       # 包含静态文件 (CSS/JS)
    '--paths=.',                          # 将当前目录添加到导入搜索路径
    '--hidden-import=flask',              # 确保 Flask 被打包
    '--hidden-import=yaml',               # 确保 PyYAML 被打包
    '--clean',
    '--distpath=dist',
]

print("开始构建过程...")
PyInstaller.__main__.run(args)
print("构建完成。可执行文件在 'dist' 文件夹中。")
