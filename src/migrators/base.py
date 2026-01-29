from abc import ABC, abstractmethod
import os

class BaseMigrator(ABC):
    def __init__(self, input_path, output_path):
        self.input_path = input_path
        self.output_path = output_path

    @abstractmethod
    def migrate(self):
        """
        执行资源迁移过程。
        """
        pass
