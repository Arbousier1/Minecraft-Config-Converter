import os
import json
from .base import BaseConverter
from src.migrators.ia_to_ce import IAMigrator

class IAConverter(BaseConverter):
    def __init__(self):
        super().__init__()
        self.ce_config = {
            "items": {},
            "equipments": {},
            "templates": {},
            "categories": {}
        }
        self.ia_resourcepack_root = None
        self.ce_resourcepack_root = None
        self.generated_models = {} # 存储需要生成的模型

    def set_resource_paths(self, ia_root, ce_root):
        self.ia_resourcepack_root = ia_root
        self.ce_resourcepack_root = ce_root

    def save_config(self, output_dir):
        """
        保存转换后的配置到输出目录中的多个文件。
        结构:
        output_dir/
          items.yml       (物品, 模板)
          armor.yml       (物品 - 护甲类型, 装备)
          categories.yml  (分类)
        """
        # 如果目录不存在则创建
        os.makedirs(output_dir, exist_ok=True)
        
        # 将物品分为护甲物品和其他物品
        armor_items = {}
        other_items = {}
        
        for key, value in self.ce_config["items"].items():
            is_armor = False
            if self._is_armor(value.get("material", "")):
                is_armor = True
            elif "settings" in value and "equipment" in value["settings"]:
                is_armor = True
                
            if is_armor:
                armor_items[key] = value
            else:
                other_items[key] = value

        # 1. 保存 items.yml (其他物品 + 模板)
        items_data = {}
        if self.ce_config["templates"]:
            items_data["templates"] = self.ce_config["templates"]
        if other_items:
            items_data["items"] = other_items
            
        if items_data:
            self._write_yaml_with_footer(items_data, os.path.join(output_dir, "items.yml"))

        # 2. 保存 armor.yml (护甲物品 + 装备)
        armor_data = {}
        if armor_items:
             armor_data["items"] = armor_items
        if self.ce_config["equipments"]:
             armor_data["equipments"] = self.ce_config["equipments"]
             
        if armor_data:
            self._write_yaml_with_footer(armor_data, os.path.join(output_dir, "armor.yml"))

        # 3. 保存 categories.yml (分类)
        if self.ce_config["categories"]:
            cat_data = {"categories": self.ce_config["categories"]}
            self._write_yaml_with_footer(cat_data, os.path.join(output_dir, "categories.yml"))

        # 如果设置了路径，触发资源迁移
        if self.ia_resourcepack_root and self.ce_resourcepack_root:
            migrator = IAMigrator(
                self.ia_resourcepack_root, 
                self.ce_resourcepack_root, 
                self.namespace
            )
            migrator.migrate()
            
        # 写入生成的模型
        if self.ce_resourcepack_root and self.generated_models:
            models_root = os.path.join(self.ce_resourcepack_root, "assets", self.namespace, "models")
            for rel_path, content in self.generated_models.items():
                full_path = os.path.join(models_root, rel_path)
                os.makedirs(os.path.dirname(full_path), exist_ok=True)
                with open(full_path, 'w', encoding='utf-8') as f:
                    json.dump(content, f, indent=4)

    def convert(self, ia_data, namespace=None):
        if namespace:
            self.namespace = namespace
        elif "info" in ia_data and "namespace" in ia_data["info"]:
            self.namespace = ia_data["info"]["namespace"]

        # 转换物品
        if "items" in ia_data:
            self._convert_items(ia_data["items"])

        # 转换装备 (旧方式)
        if "equipments" in ia_data:
            self._convert_equipments(ia_data["equipments"])
            
        # 转换 armors_rendering (新方式)
        if "armors_rendering" in ia_data:
            self._convert_armors_rendering(ia_data["armors_rendering"])

        # 转换分类
        if "categories" in ia_data:
            self._convert_categories(ia_data["categories"])

        return self.ce_config

    def _convert_items(self, items_data):
        for item_key, item_data in items_data.items():
            self._convert_item(item_key, item_data)

    def _convert_categories(self, categories_data):
        """
        将 ItemsAdder 分类转换为 CraftEngine 分类
        """
        for cat_key, cat_data in categories_data.items():
            ce_cat_id = f"{self.namespace}:{cat_key}"
            
            # 映射物品列表
            # IA 物品可能有也可能没有命名空间。如果没有，假设为当前命名空间。
            ia_items = cat_data.get("items", [])
            ce_items = []
            for item in ia_items:
                if ":" in item:
                    # 检查是否匹配当前命名空间，否则保持原样或调整
                    if item.startswith(f"{self.namespace}:"):
                        ce_items.append(item)
                    else:
                        # 如果 IA 配置有不同于我们转换的显式命名空间，
                        # 我们可能需要小心。但通常只是 'id' 或 'namespace:id'。
                        # 目前，如果它有命名空间，就信任输入。
                        ce_items.append(item)
                else:
                    ce_items.append(f"{self.namespace}:{item}")

            # 映射图标
            icon = cat_data.get("icon", "minecraft:stone")
            if ":" not in icon:
                 icon = f"{self.namespace}:{icon}"

            ce_category = {
                "name": f"<!i>{cat_data.get('name', cat_key)}",
                "lore": [
                    "<!i><gray>该配置由 <#FFFF00>MMC TOOL</#FFFF00> 生成",
                    "<!i><gray>闲鱼店铺: <#FFFF00>快乐售货铺</#FFFF00>",
                    "<!i><dark_gray>感谢您的支持！</dark_gray>"
                ],
                "priority": 1, # 默认值
                "icon": icon,
                "list": ce_items,
                "hidden": not cat_data.get("enabled", True)
            }
            
            self.ce_config["categories"][ce_cat_id] = ce_category

    def _convert_item(self, key, data):
        ce_id = f"{self.namespace}:{key}"
        
        resource = data.get("resource", {})
        material = resource.get("material", "STONE")
        display_name = data.get("display_name", key)
        
        ce_item = {
            "material": material,
            "data": {
                "item-name": self._format_display_name(display_name, data)
            }
        }

        # 根据材质或行为处理特定类型
        behaviours = data.get("behaviours", {})
        
        if self._is_armor(material, data):
            self._handle_armor(ce_item, data)
        elif behaviours.get("furniture"):
            self._handle_furniture(ce_item, data, ce_id)
        elif self._is_complex_item(material):
            self._handle_complex_item(ce_item, key, data, material)
        elif behaviours.get("hat"):
             ce_item["data"]["equippable"] = {"slot": "head"}
             self._handle_generic_model(ce_item, resource)
        else:
            self._handle_generic_model(ce_item, resource)

        self.ce_config["items"][ce_id] = ce_item

    def _is_armor(self, material, ia_data=None):
        suffixes = ["_HELMET", "_CHESTPLATE", "_LEGGINGS", "_BOOTS"]
        if any(material.endswith(s) for s in suffixes):
            return True
            
        if ia_data:
            if "specific_properties" in ia_data and "armor" in ia_data["specific_properties"]:
                return True
            if "equipment" in ia_data:
                return True
                
        return False

    def _handle_armor(self, ce_item, ia_data):
        equipment_id = None
        slot = "head"

        # 检查旧版 equipment
        if "equipment" in ia_data:
            equipment_id = ia_data["equipment"].get("id")
            
        # 检查 specific_properties armor
        if not equipment_id and "specific_properties" in ia_data:
            armor_props = ia_data["specific_properties"].get("armor", {})
            equipment_id = armor_props.get("custom_armor")
            if "slot" in armor_props:
                slot = armor_props["slot"]

        # 如果需要，从材质推断槽位 (尽管 specific_properties 通常会设置它)
        material = ce_item["material"]
        if material.endswith("_CHESTPLATE"): slot = "chest"
        elif material.endswith("_LEGGINGS"): slot = "legs"
        elif material.endswith("_BOOTS"): slot = "feet"
        
        if equipment_id:
            # 如果材质是默认的 STONE，更新材质以确保其可穿戴
            if ce_item["material"] == "STONE":
                if slot == "head": ce_item["material"] = "LEATHER_HELMET"
                elif slot == "chest": ce_item["material"] = "LEATHER_CHESTPLATE"
                elif slot == "legs": ce_item["material"] = "LEATHER_LEGGINGS"
                elif slot == "feet": ce_item["material"] = "LEATHER_BOOTS"

            ce_item["settings"] = {
                "equipment": {
                    "asset-id": f"{self.namespace}:{equipment_id}",
                    "slot": slot
                }
            }
        
        # 如果存在则添加模型
        self._handle_generic_model(ce_item, ia_data.get("resource", {}))

    def _handle_furniture(self, ce_item, ia_data, ce_id):
        furniture_data = ia_data.get("behaviours", {}).get("furniture", {})
        sit_data = ia_data.get("behaviours", {}).get("furniture_sit")
        
        ce_item["behavior"] = {
            "type": "furniture_item",
            "furniture": {
                "settings": {
                    "item": ce_id,
                    "sounds": {
                        "break": "minecraft:block.stone.break",
                        "place": "minecraft:block.stone.place"
                    }
                },
                "loot": {
                    "template": "default:loot_table/furniture",
                    "arguments": {
                        "item": ce_id
                    }
                }
            }
        }
        
        # 处理放置规则 (Placement)
        placement = {}
        placeable_on = furniture_data.get("placeable_on", {})
        
        # 如果未指定，默认为地面
        if not placeable_on:
            placeable_on = {"floor": True}

        if placeable_on.get("floor"):
            placement["ground"] = self._create_placement_block(ce_id, furniture_data, "ground", sit_data)
        if placeable_on.get("walls"):
            placement["wall"] = self._create_placement_block(ce_id, furniture_data, "wall", sit_data)
        if placeable_on.get("ceiling"):
            placement["ceiling"] = self._create_placement_block(ce_id, furniture_data, "ceiling", sit_data)
            
        ce_item["behavior"]["furniture"]["placement"] = placement

        self._handle_generic_model(ce_item, ia_data.get("resource", {}))

    def _create_placement_block(self, ce_id, furniture_data, placement_type, sit_data=None):
        """
        创建家具放置块 (ground, wall, ceiling) 的通用配置
        """
        # 计算 Translation
        # CE 的 translation 通常是高度的一半，使模型底部对齐地面
        height = 1
        width = 1
        length = 1
        
        if "hitbox" in furniture_data:
             hitbox = furniture_data["hitbox"]
             height = hitbox.get("height", 1)
             width = hitbox.get("width", 1)
             length = hitbox.get("length", 1)
        
        translation_y = height / 2.0
        
        block_config = {
            "loot-spawn-offset": "0,0.4,0",
            "rules": {
                "rotation": "ANY",
                "alignment": "ANY"
            },
            "elements": [
                {
                    "item": ce_id,
                    "display-transform": "NONE",
                    "shadow-radius": 0.4,
                    "shadow-strength": 0.5,
                    "billboard": "FIXED",
                    "translation": f"0,{translation_y},0"
                }
            ]
        }
        
        # 处理 Hitbox
        # 官方逻辑: 将家具拆分为多个 1x1 的 Shulker 碰撞箱
        if "hitbox" in furniture_data:
            ia_hitbox = furniture_data["hitbox"]
            is_solid = furniture_data.get("solid", True)
            
            # 获取 IA 偏移
            w_offset = ia_hitbox.get("width_offset", 0)
            h_offset = ia_hitbox.get("height_offset", 0)
            l_offset = ia_hitbox.get("length_offset", 0)
            
            hitboxes = []
            
            # 只有 solid 的家具才生成 shulker 碰撞箱矩阵
            # 非 solid 的家具可能需要 interaction 类型 (暂不处理或生成单个)
            
            # 特殊情况: 如果是可坐的 (sit_data)，使用 interaction 类型以支持座位
            # 并且根据 solid 属性设置 blocks-building
            if sit_data:
                # 提取座位高度
                # IA 默认为 0 (相对于家具底部) ? 
                # 通常 IA sit_height 是相对于地面的高度 (e.g. 0.8)
                # CE translation_y = height / 2.0 (中心)
                # CE interaction hitbox 是相对于家具中心的吗?
                # 假设 CE seats 坐标是相对于 hitbox 的
                
                # 简单的转换逻辑:
                # CE seat Y = IA sit_height - 0.85 (经验值, 微调)
                # 或者尝试 IA sit_height - 0.5 (如果 Y 轴原点在中心)
                
                ia_sit_height = sit_data.get("sit_height", 0.5)
                # 尝试对齐示例: 0.8 -> -0.05.  diff = 0.85
                ce_seat_y = ia_sit_height - 0.85
                
                hitboxes.append({
                    "position": "0,0,0",
                    "type": "interaction",
                    "blocks-building": is_solid,
                    "width": width, # 使用家具定义的宽度
                    "height": height,
                    "interactive": True,
                    "seats": [f"0,{ce_seat_y:g},0"]
                })
            
            elif is_solid:
                # 遍历体积生成 1x1 碰撞箱
                # width -> x, height -> y, length -> z
                # 确保转换为整数循环范围
                w_range = int(round(width))
                h_range = int(round(height))
                l_range = int(round(length))
                
                # 如果尺寸小于 1，至少生成 1 个
                w_range = max(1, w_range)
                h_range = max(1, h_range)
                l_range = max(1, l_range)

                for y in range(h_range):
                    for x in range(w_range):
                        for z in range(l_range):
                            # 计算相对中心的位置
                            # 居中逻辑: (i - (count - 1) / 2)
                            
                            rel_x = x - (w_range - 1) / 2.0
                            rel_y = y # 高度通常从底部开始，所以不居中，直接向上延伸? 
                            # 检查 Big Cupboard: Y=0,1,2. 确实是从 0 开始递增.
                            
                            rel_z = z - (l_range - 1) / 2.0
                            
                            # 应用偏移
                            final_x = rel_x + w_offset
                            final_y = rel_y + h_offset
                            final_z = rel_z + l_offset
                            
                            # Shulker 位置应该是整数 (格式化去除 .0)
                            # 使用 :g 可以自动去除不必要的浮点
                            pos_str = f"{final_x:g},{final_y:g},{final_z:g}"
                            
                            hitboxes.append({
                                "position": pos_str,
                                "type": "shulker",
                                "blocks-building": True,
                                "interactive": True
                            })
            else:
                # 非实体，生成一个交互框
                hitboxes.append({
                    "position": "0,0,0",
                    "type": "interaction",
                    "blocks-building": False,
                    "width": width,
                    "height": height,
                    "interactive": True
                })

            block_config["hitboxes"] = hitboxes
            
        return block_config

    def _is_complex_item(self, material):
        return material in ["BOW", "CROSSBOW", "FISHING_ROD", "SHIELD"]

    def _handle_complex_item(self, ce_item, key, ia_data, material):
        # 为此物品创建一个模板
        template_id = f"models:{self.namespace}_{key}_model"
        
        # 在真实场景中，我们会扫描 JSON 文件以查找谓词。
        # 目前，我们基于材质类型和标准 IA 命名约定生成标准模板。
        
        template_def = {}
        args = {}
        
        resource = ia_data.get("resource", {})
        base_model_path = resource.get("model_path", "")
        
        if material == "BOW":
            template_def = {
                "type": "minecraft:condition",
                "property": "minecraft:using_item",
                "on-false": {"type": "minecraft:model", "path": "${bow_model}"},
                "on-true": {
                    "type": "minecraft:range_dispatch",
                    "property": "minecraft:use_duration",
                    "scale": 0.05,
                    "entries": [
                        {"threshold": 0.65, "model": {"type": "minecraft:model", "path": "${bow_pulling_1_model}"}},
                        {"threshold": 0.9, "model": {"type": "minecraft:model", "path": "${bow_pulling_2_model}"}}
                    ],
                    "fallback": {"type": "minecraft:model", "path": "${bow_pulling_0_model}"}
                }
            }
            # 推断路径
            args["bow_model"] = f"{self.namespace}:item/{base_model_path}"
            args["bow_pulling_0_model"] = f"{self.namespace}:item/{base_model_path}_0"
            args["bow_pulling_1_model"] = f"{self.namespace}:item/{base_model_path}_1"
            args["bow_pulling_2_model"] = f"{self.namespace}:item/{base_model_path}_2"

        elif material == "CROSSBOW":
            # 简化的弩模板
            template_def = {
                "type": "minecraft:condition",
                "property": "minecraft:using_item",
                "on-false": {
                    "type": "minecraft:select",
                    "property": "minecraft:charge_type",
                    "cases": [
                        {"when": "arrow", "model": {"type": "minecraft:model", "path": "${arrow_model}"}},
                        {"when": "rocket", "model": {"type": "minecraft:model", "path": "${firework_model}"}}
                    ],
                    "fallback": {"type": "minecraft:model", "path": "${model}"}
                },
                "on-true": {
                     "type": "minecraft:range_dispatch",
                     "property": "minecraft:crossbow/pull",
                     "entries": [
                         {"threshold": 0.58, "model": {"type": "minecraft:model", "path": "${pulling_1_model}"}},
                         {"threshold": 1.0, "model": {"type": "minecraft:model", "path": "${pulling_2_model}"}}
                     ],
                     "fallback": {"type": "minecraft:model", "path": "${pulling_0_model}"}
                }
            }
            args["model"] = f"{self.namespace}:item/{base_model_path}"
            args["arrow_model"] = f"{self.namespace}:item/{base_model_path}_charged"
            args["firework_model"] = f"{self.namespace}:item/{base_model_path}_firework"
            args["pulling_0_model"] = f"{self.namespace}:item/{base_model_path}_0"
            args["pulling_1_model"] = f"{self.namespace}:item/{base_model_path}_1"
            args["pulling_2_model"] = f"{self.namespace}:item/{base_model_path}_2"
            
        elif material == "SHIELD":
            template_def = {
                "type": "minecraft:condition",
                "property": "minecraft:using_item",
                "on-false": {"type": "minecraft:model", "path": "${shield_model}"},
                "on-true": {"type": "minecraft:model", "path": "${shield_blocking_model}"}
            }
            args["shield_model"] = f"{self.namespace}:item/{base_model_path}"
            args["shield_blocking_model"] = f"{self.namespace}:item/{base_model_path}_blocking"
            
        elif material == "FISHING_ROD":
             template_def = {
                "type": "minecraft:condition",
                "property": "minecraft:fishing_rod/cast",
                "on-false": {"type": "minecraft:model", "path": "${path}"},
                "on-true": {"type": "minecraft:model", "path": "${cast_path}"}
            }
             args["path"] = f"{self.namespace}:item/{base_model_path}"
             args["cast_path"] = f"{self.namespace}:item/{base_model_path}_cast"

        # 注册模板
        self.ce_config["templates"][template_id] = template_def
        
        # 分配给物品
        ce_item["model"] = {
            "template": template_id,
            "arguments": args
        }

    def _handle_generic_model(self, ce_item, resource):
        model_path = resource.get("model_path")
        
        # 情况 1: 显式模型路径
        if model_path:
            ce_item["model"] = {
                "type": "minecraft:model",
                "path": f"{self.namespace}:item/{model_path}"
            }
        
        # 情况 2: 从纹理生成模型
        elif resource.get("generate") is True and resource.get("textures"):
            # 使用第一个纹理路径作为模型路径的基础
            # IA: textures: [path/to/texture] -> 通常意味着在 assets/namespace/models/path/to/texture.json 生成模型
            # CE: 我们将指向 namespace:item/path/to/texture
            
            texture_path = resource["textures"][0]
            # 如果存在 .png 扩展名则移除 (IA 这里通常没有 .png，但为了安全起见)
            if texture_path.endswith(".png"):
                texture_path = texture_path[:-4]
                
            ce_item["model"] = {
                "type": "minecraft:model",
                "path": f"{self.namespace}:item/{texture_path}"
            }

            # 注册此模型以进行生成
            # CE 引用: namespace:item/texture_path
            # 文件路径: assets/namespace/models/item/texture_path.json
            # 纹理引用: namespace:item/texture_path (假设迁移器将其移动到 item/)
            
            model_key = f"item/{texture_path}.json"
            self.generated_models[model_key] = {
                "parent": "minecraft:item/generated",
                "textures": {
                    "layer0": f"{self.namespace}:item/{texture_path}"
                }
            }

    def _convert_equipments(self, equipments_data):
        for eq_key, eq_data in equipments_data.items():
            ce_eq_id = f"{self.namespace}:{eq_key}"
            
            # 映射 IA 图层到 CE Humanoid 图层
            ce_eq = {
                "type": "component"
            }
            
            if "layer_1" in eq_data:
                ce_eq["humanoid"] = f"{self.namespace}:{eq_data['layer_1']}"
            if "layer_2" in eq_data:
                ce_eq["humanoid-leggings"] = f"{self.namespace}:{eq_data['layer_2']}"
                
            self.ce_config["equipments"][ce_eq_id] = ce_eq

    def _convert_armors_rendering(self, armors_rendering_data):
        """
        将 IA 的 'armors_rendering' 转换为 CraftEngine 的 'equipments'。
        """
        for armor_name, armor_data in armors_rendering_data.items():
            ce_key = f"{self.namespace}:{armor_name}"
            
            ce_entry = {
                "type": "component"
            }
            
            # 映射 layer_1 -> humanoid
            if "layer_1" in armor_data:
                # IA: armor/layer_1
                # CE: namespace:armor/layer_1 (将被解析为纹理)
                # 我们需要为 CE 引用前置命名空间
                layer_1_path = armor_data["layer_1"]
                # 如果路径以 .png 结尾，移除它 (如果引用纹理资源，CE 引用配置中通常没有扩展名，等等..
                # 实际上 CE 纹理引用通常是 namespace:path/to/texture)
                if layer_1_path.endswith(".png"):
                     layer_1_path = layer_1_path[:-4]
                
                ce_entry["humanoid"] = f"{self.namespace}:{layer_1_path}"

            # 映射 layer_2 -> humanoid-leggings
            if "layer_2" in armor_data:
                layer_2_path = armor_data["layer_2"]
                if layer_2_path.endswith(".png"):
                     layer_2_path = layer_2_path[:-4]
                ce_entry["humanoid-leggings"] = f"{self.namespace}:{layer_2_path}"

            self.ce_config["equipments"][ce_key] = ce_entry

    def _format_display_name(self, name, data=None):
        # 基本 MiniMessage 转换
        # IA 经常使用传统颜色代码，或者纯文本。
        # 对于此原型，如果尚未格式化，我们将用 CE 风格包装它
        if "&" in name or "§" in name:
            # 替换传统代码 (简化)
            name = name.replace("&", "§")
            # 如果需要，将常用代码映射到 MiniMessage 标签，或者 CE 可能支持传统代码？
            # CE 通常使用 MiniMessage <color>。
            # 目前，我们只假设简单文本需要包装。
            pass
            
        # 示例格式: <!i><#FFCF20>Name
        # 如果提供了 data，我们可以检查是否有自定义颜色配置 (在此示例中，我们硬编码 EliteCreatures 风格)
        # 或者仅仅将其作为默认值
        default_color = "<white>"
        if data and "elitecreatures" in self.namespace:
             default_color = "<#FFCF20>"
             
        return f"<!i>{default_color}{name}"
