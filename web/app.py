from flask import Flask, render_template, request, send_file, jsonify
import os
import shutil
import zipfile
import uuid
import re
import time
import yaml

# 閻庣數鍘ч崣鍡涘冀缁嬭法濡囬梺顐ｆ缁?
import sys
import webbrowser
from threading import Timer

if getattr(sys, 'frozen', False):
    BASE_DIR = os.path.dirname(sys.executable)
else:
    BASE_DIR = os.path.dirname(os.path.abspath(__file__))
# 閻忓繐妫濋妴宥夋儎椤旂晫澹岄柣鈺婂枛缂嶅秴菐鐠囨彃顫ｉ柛?python path
sys.path.append(os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
from src.converters.ia_to_ce import IAConverter
from src.converters.nexo_to_ce import NexoConverter
from src.analyzer import PackageAnalyzer
from src.utils.yaml_loader import safe_load_yaml

app = Flask(__name__)
app.config['UPLOAD_FOLDER'] = os.path.join(BASE_DIR, 'temp_uploads')
app.config['OUTPUT_FOLDER'] = os.path.join(BASE_DIR, 'temp_output')
app.config['MAX_CONTENT_LENGTH'] = 500 * 1024 * 1024  # 500MB 闂傚嫭鍔曢崺?

# 闁衡偓椤栨稑鐦柣銊ュ瑜板啯绂掔捄鍝勭仚閻?
SUPPORTED_PLUGINS = [
    {"id": "ItemsAdder", "name": "ItemsAdder", "icon": "/static/images/itemsadder.webp"},
    {"id": "Nexo", "name": "Nexo", "icon": "/static/images/nexo.webp"},
    {"id": "Oraxen", "name": "Oraxen", "icon": "/static/images/oraxen.webp"},
    {"id": "CraftEngine", "name": "CraftEngine", "icon": "/static/images/craftengine.webp"},
    {"id": "MythicCrucible", "name": "MythicCrucible", "icon": "/static/images/mythiccrucible.webp"}
    # {"id": "HMCCosmetics", "name": "HMCCosmetics", "icon": "妫ｅ啯鍟?}
]

# 缁绢収鍠曠换姘▔鐎涙ɑ顦ч柣鈺婂枛缂嶅秶鈧稒锚濠€?
os.makedirs(app.config['UPLOAD_FOLDER'], exist_ok=True)
os.makedirs(app.config['OUTPUT_FOLDER'], exist_ok=True)

@app.route('/')
def index():
    return render_template('index.html')

@app.route('/api/analyze', methods=['POST'])
def analyze():
    if 'file' not in request.files:
        return jsonify({'error': '婵炲备鍓濆﹢渚€寮ㄧ捄鍝勭厒闁哄倸娲ｅ▎?}), 400

    file = request.files['file']
    if file.filename == '':
        return jsonify({'error': '闁哄牜浜埀顒€顦扮€氥劑寮崶锔筋偨'}), 400

    if file:
        session_id = str(uuid.uuid4())
        session_upload_dir = os.path.join(app.config['UPLOAD_FOLDER'], session_id)
        os.makedirs(session_upload_dir, exist_ok=True)

        try:
            filename = file.filename
            file_path = os.path.join(session_upload_dir, filename)
            file.save(file_path)

            extract_dir = os.path.join(session_upload_dir, "extracted")
            if filename.endswith('.zip'):
                with zipfile.ZipFile(file_path, 'r') as zip_ref:
                    zip_ref.extractall(extract_dir)
            else:
                return jsonify({'error': '閻犲洩娓圭粭鍌涘?.zip 闁哄倸娲ｅ▎?}), 400

            # 閺夆晜鍔橀、鎴﹀礆閸℃鈧?
            analyzer = PackageAnalyzer(extract_dir)
            report = analyzer.analyze()

            # 闁哄秷顫夊畵浣肝涢埀顒€霉鐎ｎ亜鐓傞柣銊ュ閻楃顕ｈ箛鏇椻偓妯尖偓瑙勮壘瑜版煡鎮介妸褎鐣遍柣鈺婂枟閻栵綁寮介悡搴ｇ
            # 闂侇偅妲掔欢顐︽晬?
            # 1. 閻犲洤妫楅崺鍡椻攦閹邦厾澹愮€?(闁告瑯鍨甸崗姗€宕犻崨顓熷創濠㈣埖鐭柌?
            # 2. 濠碘€冲€归悘澶愬礌閸涱厽鍎?ItemsAdder -> 闁稿繋娴囬蹇旀姜椤戣儻绀?CraftEngine (闂傚嫨鍊濆顏勵啅閹绘帒鐦堕柛?CraftEngine)
            # 3. 濠碘€冲€归悘澶愬礌閸涱厽鍎?CraftEngine -> 闁哄棗鍊瑰Λ銈嗘姜椤掍礁搴?(闁瑰瓨鐗曢崢鎴犳媼濮濆本绁☉?ItemsAdder)
            # 4. 濠碘€冲€归悘澶愬礌閸涱厽鍎?Nexo -> 闁哄棗鍊瑰Λ銈嗘姜椤掍礁搴?

            detected_formats = report["formats"]
            available_targets = []
            warnings = []

            if "ItemsAdder" in detected_formats:
                if "CraftEngine" in detected_formats:
                    warnings.append("婵☆偀鍋撴繛鏉戭儏閸╁矂宕犻崨顒冨幀鐎瑰憡褰冪€垫﹢宕?CraftEngine 闂佹澘绉堕悿鍡涘Υ閸屾繃绁柟骞垮灩瑜版煡鎳楅幋鎺旂獥閻熸洖妫涘ú濠囧箣閺嶏箓鐛撻柣銏㈠枎閸熻法绮ｆ担纰樺亾?)
                if "CraftEngine" not in available_targets:
                    available_targets.append("CraftEngine")

            if "Nexo" in detected_formats:
                if "CraftEngine" in detected_formats:
                    warnings.append("婵☆偀鍋撴繛鏉戭儏閸╁矂宕犻崨顒冨幀鐎瑰憡褰冪€垫﹢宕?CraftEngine 闂佹澘绉堕悿鍡涘Υ閸屾繃绁柟骞垮灩瑜版煡鎳楅幋鎺旂獥閻熸洖妫涘ú濠囧箣閺嶏箓鐛撻柣銏㈠枎閸熻法绮ｆ担纰樺亾?)
                if "CraftEngine" not in available_targets:
                    available_targets.append("CraftEngine")

            if "CraftEngine" in detected_formats:
                 # 闁哄牜浜濆鐢稿绩椤栨稑鐦?CE -> IA
                 pass

            report["source_formats"] = detected_formats # 闁衡偓閻熺増鍊冲ù鐘劚瀵粙寮伴悩宸Щ闁?
            report["available_targets"] = available_targets
            report["warnings"] = warnings
            report["filename"] = filename
            report["supported_plugins"] = SUPPORTED_PLUGINS

            return jsonify({
                'status': 'success',
                'report': report,
                'session_id': session_id
            })

        except Exception as e:
            return jsonify({'error': str(e)}), 500

@app.route('/api/convert', methods=['POST'])
def convert():
    # 闁衡偓椤栨稑鐦☉鎾卞€楅～鎺懳熼垾宕囩闁?
    # 1. 濞磋偐濮风划娲儍閸曨厽绾柟鎭掑劙缁楀倹瀵奸悩铏€ù鐘烘硾閼荤喐娼浣稿簥 (濞ｅ洦绻冪€垫棃宕楅悡搴晣)
    # 2. 闁规亽鍎辫ぐ?session_id (濞?/api/analyze 闁兼儳鍢茶ぐ? 閺夆晜绋栭、鎴炴姜椤掍礁搴?

    session_id = request.form.get('session_id')
    target_format = request.form.get('target_format', 'CraftEngine') # 濮掓稒顭堥?CE
    source_format = request.form.get('source_format') # 闁哄倹婢橀·? 闁哄嫬娴烽垾妯衡攦閹邦厾澹愮€?

    if session_id:
        # 濞达綀娉曢弫銈咁啅閹绘帞鎽犻柛锔哄妿濞堟垶瀵煎宕囨▓
        session_upload_dir = os.path.join(app.config['UPLOAD_FOLDER'], session_id)
        extract_dir = os.path.join(session_upload_dir, "extracted")
        if not os.path.exists(extract_dir):
            return jsonify({'error': '濞村吋淇洪惁钘夘啅閼煎墎绠栭柡鍫㈠枑閸ㄣ劍绋夊鍛憼闁?}), 400

        session_output_dir = os.path.join(app.config['OUTPUT_FOLDER'], session_id)
        os.makedirs(session_output_dir, exist_ok=True)

    elif 'file' in request.files:
        # 濞磋偐濮风划鍝勎熼垾宕囩
        file = request.files['file']
        if file.filename == '':
            return jsonify({'error': '闁哄牜浜埀顒€顦扮€氥劑寮崶锔筋偨'}), 400

        session_id = str(uuid.uuid4())
        session_upload_dir = os.path.join(app.config['UPLOAD_FOLDER'], session_id)
        session_output_dir = os.path.join(app.config['OUTPUT_FOLDER'], session_id)
        os.makedirs(session_upload_dir, exist_ok=True)
        os.makedirs(session_output_dir, exist_ok=True)

        filename = file.filename
        file_path = os.path.join(session_upload_dir, filename)
        file.save(file_path)

        extract_dir = os.path.join(session_upload_dir, "extracted")
        if filename.endswith('.zip'):
            with zipfile.ZipFile(file_path, 'r') as zip_ref:
                zip_ref.extractall(extract_dir)
        else:
            return jsonify({'error': '閻犲洩娓圭粭鍌涘?.zip 闁哄倸娲ｅ▎?}), 400
    else:
        return jsonify({'error': '闁哄啰濮甸弲銉╂儍閸曨噮鍤炴慨?}), 400

    try:
        if target_format == "CraftEngine":
            if source_format == "Nexo":
                return _convert_nexo_to_ce(extract_dir, session_output_dir, session_upload_dir, target_format)
            else:
                # 濮掓稒顭堥缁樼▔?ItemsAdder 闁瑰瓨鐗楀Ο澶婎嚕韫囨柨鐦归悗?
                return _convert_ia_to_ce(extract_dir, session_output_dir, session_upload_dir, target_format)

        return jsonify({'error': f'濞戞挸绉甸弫顕€骞愭担鐑樼暠闁烩晩鍠楅悥锝夊冀閻撳海纭€: {target_format}'}), 400

    except Exception as e:
        import traceback
        traceback.print_exc()
        return jsonify({'error': str(e)}), 500

def _convert_nexo_to_ce(extract_dir, session_output_dir, session_upload_dir, target_format):
    # 1. 闁规鍋呭?Nexo 闂佹澘绉堕悿鍡涘椽瀹€鍐偒婵?
    nexo_items_configs = []
    nexo_resourcepack_path = None

    # 閻忓繑绻嗛惁顖炲箥閹冪厒 Nexo 闁哄秴婀卞ú鎷屻亹?
    scan_root = extract_dir
    for root, dirs, files in os.walk(extract_dir):
        if "Nexo" in dirs:
            scan_root = os.path.join(root, "Nexo")
            break
        elif "nexo" in dirs:
             scan_root = os.path.join(root, "nexo")
             break

    # 闁规鍋呭鍧楁煀瀹ュ洨鏋傞柛婊冪焷缁侇偄鈹?
    for root, dirs, files in os.walk(scan_root):
        # 閻犙冨缁噣宕犻崨顕呮⒕婵?
        if "pack" in dirs and nexo_resourcepack_path is None:
             nexo_resourcepack_path = os.path.join(root, "pack")
        elif "assets" in dirs and nexo_resourcepack_path is None:
             nexo_resourcepack_path = root

        # 闂佹澘绉堕悿鍡涘棘閸ワ附顐芥俊顐熷亾婵?
        for f in files:
            if f.endswith((".yml", ".yaml")):
                full_path = os.path.join(root, f)
                # 缂佺姭鍋撻柛妤佹礉缁诲啫顭ㄩ妶蹇曠闂侇剙鐏濋崢銈夊礉閻樼儤绁伴梻鍫㈠仱閸樸倗绱?
                if "config.yml" in f: continue
                nexo_items_configs.append(full_path)

    if not nexo_items_configs:
         return jsonify({'error': '闁哄牜浜ｉ崗姗€骞嶉幆褍鐓?Nexo 闂佹澘绉堕悿鍡涘棘閸ワ附顐?}), 400

    # 2. 閺夆晜鍔橀、鎴炴姜椤掍礁搴?
    # 闁告垵妫楅ˇ顒勫川閽樺鍊崇紒灞炬そ濡?
    user_namespace = request.form.get('namespace')

    if user_namespace and re.match(r'^[0-9a-z_.-]+$', user_namespace):
        # 闁活潿鍔嶉崺娑㈠箰閸パ呮毎濞存粌妫楅幊锟犲触瀹ュ洠鏁勯梻鍌濇彧缁辨繈宕ラ崼婵婂珯闁圭鍋撻柡鍫濐樀閸樸倗绱?
        converter = NexoConverter()
        merged_data = {}
        for config_path in nexo_items_configs:
            data = safe_load_yaml(config_path)
            if isinstance(data, dict):
                 merged_data.update(data)

        namespace = user_namespace
        ce_output_base = os.path.join(session_output_dir, "CraftEngine", "resources", namespace)
        ce_config_dir = os.path.join(ce_output_base, "configuration", "items", namespace)
        ce_res_dir = os.path.join(ce_output_base, "resourcepack")

        if nexo_resourcepack_path:
            converter.set_resource_paths(nexo_resourcepack_path, ce_res_dir)

        converter.convert(merged_data, namespace=namespace)
        converter.save_config(ce_config_dir)

    else:
        # 闁活潿鍔嶉崺娑㈠嫉椤忓懎鐦归悗瑙勮壘閹筹繝宕ュ鍥ｆ晞闂傚倽鎻槐婵囨媴鐠恒劍鏆忛柡鍌氭矗濞嗐垽宕ュ鍕▕濞戞挸鎼幊锟犲触瀹ュ洠鏁勯梻?
        for config_path in nexo_items_configs:
            data = safe_load_yaml(config_path)
            if not isinstance(data, dict):
                continue

            # 濞寸姴瀛╅弸鍐╃鐠虹儤鍊抽柤鎯у槻瑜板洭宕ㄩ挊澶嬪€崇紒灞炬そ濡?
            filename = os.path.basename(config_path)
            namespace = os.path.splitext(filename)[0]
            # 缂佺姭鍋撻柛妤佹礈濞堟垿宕ㄩ挊澶嬪€崇紒灞炬そ濡灝銆掗崨顖涘€?
            namespace = re.sub(r'[^0-9a-z_.-]', '_', namespace.lower())

            # 婵絽绻嬮柌婊堝棘閸ワ附顐介柣娆樺墰閻濇稒娼浣稿簥
            converter = NexoConverter()

            ce_output_base = os.path.join(session_output_dir, "CraftEngine", "resources", namespace)
            ce_config_dir = os.path.join(ce_output_base, "configuration", "items", namespace)
            ce_res_dir = os.path.join(ce_output_base, "resourcepack")

            if nexo_resourcepack_path:
                converter.set_resource_paths(nexo_resourcepack_path, ce_res_dir)

            converter.convert(data, namespace=namespace)
            converter.save_config(ce_config_dir)

    return _package_and_respond(session_output_dir, session_upload_dir, target_format)

def _convert_ia_to_ce(extract_dir, session_output_dir, session_upload_dir, target_format):
    # 3. 閻庤鐭紞鍛存煀瀹ュ洨鏋傞柛婊冪焷缁侇偄鈹?(ItemsAdder -> CraftEngine 闂侇偅妲掔欢?
    # 闁衡偓绾懐绠婚梺顐ｆ缁? 闁规鍋呭鍧楀箥閳ь剟寮?YAML 闁哄倸娲ｅ▎銏ょ嵁閼哥數澹岄柟璇″枛閸炲鈧湱顢婄换妯兼偘鐏炶棄鐎荤紒?
    ia_items_configs = []
    ia_categories_configs = []
    ia_recipes_configs = []
    ia_resourcepack_path = None

    # 0. 缁绢収鍠栭悾楣冨箥椤愶絽浼庨柡宥呮贡濞叉媽銇?
    scan_root = extract_dir
    found_ia_dir = False
    for root, dirs, files in os.walk(extract_dir):
        for d in dirs:
            if d.lower() == "itemsadder":
                scan_root = os.path.join(root, d)
                found_ia_dir = True
                break
        if found_ia_dir:
            break

    if found_ia_dir:
            print(f"Detected ItemsAdder root at: {scan_root}")

    # 缂佹鍏涚粩鎾焼瀹ュ棗顥囬柟璇查獜缁变即寮婚妷锕€顥濋梺鏉跨Ф閻ゅ棝寮崶锔筋偨闁告粌鏈悥锝夊礄閸℃氨銈繝褎鍔曠€垫绱掗幘瀵糕偓?
    for root, dirs, files in os.walk(scan_root):
        # --- 閻犙冨缁噣宕犻崨顕呮⒕婵?---
        # 濞村吋锚閸樻稓鐥?1: 闁哄嫭鍎崇槐锟犳儍?"resourcepack" 闁烩晩鍠栫紞?
        if "resourcepack" in dirs and ia_resourcepack_path is None:
            ia_resourcepack_path = os.path.join(root, "resourcepack")

        # 濞村吋锚閸樻稓鐥?2: 闁烩晛鐡ㄧ敮鎾礌閸涱厽鍎?assets 闁汇劌瀚ú鎷屻亹?
        if "assets" in dirs and ia_resourcepack_path is None:
            ia_resourcepack_path = root

        # 濞村吋锚閸樻稓鐥?3: 闁烩晛鐡ㄧ敮鎾礌閸涱厽鍎?models 闁?textures 闁汇劌瀚ú鎷屻亹?(闂傚牏鍋為悥锝夊礄閸℃瑧娉㈤柡?
        if "models" in dirs and "textures" in dirs and ia_resourcepack_path is None:
            ia_resourcepack_path = root

        # --- 闂佹澘绉堕悿鍡涘棘閸ワ附顐芥俊顐熷亾婵?---
        for f in files:
            if f.endswith(".yml") or f.endswith(".yaml"):
                full_path = os.path.join(root, f)
                try:
                    print(f"Scanning: {full_path}")
                    data = safe_load_yaml(full_path)
                    if not data:
                        continue

                    # 婵☆偀鍋撻柡灞诲劚閸櫻囨煥椤旂虎鍔柛?
                    if "items" in data or "equipments" in data or "armors_rendering" in data:
                        ia_items_configs.append(full_path)
                    if "categories" in data:
                        ia_categories_configs.append(full_path)
                    if "recipes" in data:
                        ia_recipes_configs.append(full_path)
                except Exception as e:
                    print(f"Error loading {full_path}: {e}")
                    continue

    # 濠碘€冲€归悘澶嬬瀹ュ棙寮撻柟鍨劤閸╁瞼鎸ч崟顒傜埍闁告牕鎷戠槐婵堜焊濠靛﹦妲搁悗鐢电帛婢?textures/models 闁汇劌瀚崺妤冪棯?(濠㈣泛瀚幃濠囨閻愬鍨奸柛鎴濇缁劑寮?
    if ia_resourcepack_path is None:
        # 濠碘€冲€归悘澶愬嫉婢舵劕甯崇紓鍐惧枟閺嬪啯绂掔拋鍦濮掓稒顭堥缁樼▔閻戞ê绲归柛娆愮墬閻楁挳鎯勯鑲╃Э
        if ia_items_configs:
            ia_resourcepack_path = extract_dir

    if not ia_items_configs:
            return jsonify({'error': '闁哄牜浜ｉ崗姗€骞嶉幆褍鐓傞柛鏍ф噹閹牓鎮ч埡浣规儌閻庤鐭粻鐔兼儍閸曨垰甯崇紓鍐惧枟閺嬪啯绂?(items/equipments)'}), 400

    # 4. 閺夆晜鍔橀、鎴炴姜椤掍礁搴?
    converter = IAConverter()

    # 闁告梻濮惧ù鍥嵁鐠虹儤鍊ゆ鐐跺煐婢у秹寮垫径灞解挅闁告繀绶氶崢銈囩磾?
    merged_items_data = {"items": {}, "equipments": {}, "armors_rendering": {}, "templates": {}, "recipes": {}, "info": {}}

    for config_path in ia_items_configs:
        data = converter.load_config(config_path)
        if not data: continue

        # 闁告艾鐗嗛懟鐔兼焻閺勫繒甯?
        if "info" in data and not merged_items_data["info"]:
            merged_items_data["info"] = data["info"] # 濞达綀娉曢弫銈夊箥閹冪厒闁汇劌瀚鍥ㄧ▔閳ь剚绋?info

        if "items" in data:
            merged_items_data.setdefault("items", {}).update(data["items"])

        if "equipments" in data:
            merged_items_data.setdefault("equipments", {}).update(data["equipments"])

        if "armors_rendering" in data:
            merged_items_data.setdefault("armors_rendering", {}).update(data["armors_rendering"])

        if "templates" in data:
            merged_items_data.setdefault("templates", {}).update(data["templates"])

    ia_data = merged_items_data

    # 濠碘€冲€归悘澶愬箥閹冪厒闁告帗鐟ユ慨鐐存姜閽樺鐎荤紒?
    if ia_categories_configs:
        merged_categories = {}
        for cat_config in ia_categories_configs:
            data = converter.load_config(cat_config)
            if data and "categories" in data:
                merged_categories.update(data["categories"])

        if merged_categories:
            ia_data["categories"] = merged_categories

    if ia_recipes_configs:
        merged_recipes = {}
        for recipe_config in ia_recipes_configs:
            data = converter.load_config(recipe_config)
            if not data:
                continue
            if "info" in data and not ia_data.get("info"):
                ia_data["info"] = data["info"]
            recipes_block = data.get("recipes")
            if not isinstance(recipes_block, dict):
                continue
            for group_key, group_data in recipes_block.items():
                if group_key not in merged_recipes:
                    merged_recipes[group_key] = {}
                if isinstance(group_data, dict):
                    merged_recipes[group_key].update(group_data)
        if merged_recipes:
            ia_data["recipes"] = merged_recipes

    # 闁告垵妫楅ˇ顒佹綇閹惧啿姣夐悹渚灠缁?
    # CraftEngine 閺夊牊鎸搁崵顓犵磼閹惧鈧? resources/<namespace>/...
    # 濞达綀娉曢弫銈夋煀瀹ュ洨鏋傚☉鎿冨幘濞堟垿宕ㄩ挊澶嬪€崇紒灞炬そ濡潡骞嬮弽顓犲笡閻犱降鍊曢埀?
    original_namespace = ia_data.get("info", {}).get("namespace", "converted")
    namespace = original_namespace

    # 婵☆偀鍋撻柡灞诲劤閺併倝骞嬮柨瀣﹂柛姘鹃檮鐎垫氨鈧鐭花锟犲川閽樺鍊崇紒灞炬そ濡?
    user_namespace = request.form.get('namespace')
    if user_namespace:
        # 濡ょ姴鐭侀惁澶愬川閽樺鍊崇紒灞炬そ濡法鎲撮崟顐㈢仧: 0-9, a-z, _, -, .
        if not re.match(r'^[0-9a-z_.-]+$', user_namespace):
            return jsonify({'error': '闁告稖妫勯幃鏇犵矚濞差亝锛熼柛鏍ф噹閹牓妫冮悙瀵搞€婇悗娑欘殘椤戜線濡撮崒娆戠煂闁稿繋娴囬蹇曚焊韫囨挸鏅搁悗娑欘殕閻︽繈濡存担瑙勬閻庢稒銇滈埀顑挎缁楀懘宕氶幒鏂挎疇闁靛棔娴囩换娑氣偓娑欘殘椤戜線宕畝鍐伆闁哄倸娲よぐ鐐哄矗閺嬵偀鍋?}), 400
        namespace = user_namespace

    # 闁绘顫夐悾鈺傚緞閸曨厽鍊為柨娑欒壘椤┭囧几濠婂棛銈繝褎鍔曠€垫绱掗幘瀵糕偓顖炲及椤栫偞濮滈柡宥呮搐閸ｎ垶鎯冮崟鍓佺闁烩晛鐡ㄧ敮鎾礌閸涱厽鍎?models/textures闁挎稑顧€缁辨繈宕氬▎鎾虫缂備礁瀚拹鐔煎冀閸パ冩珯缂備焦鎸婚悗?
    # 閺夆晜鐟╅埀顒佽壘閻栧爼宕ｉ幋鐘虫櫢闁?ia_resourcepack_path 闁圭娲ら幃婊勭閸℃鐦堕柛?models/textures 闁汇劌瀚悧鎾儎椤旇偐绉块柨娑樺缁插墽绱撻崫鍕瘜 assets/<namespace> 闁告牕鎳撻ˉ濠囨儍閸曨剙鍓伴柛?
    if ia_resourcepack_path and os.path.exists(ia_resourcepack_path):
        # 婵☆偀鍋撻柡灞诲劜閻栵綁宕欓崱娆戞尝闁哄瀚Σ鎼佸触閿曗偓閻°劑宕?
        assets_path = os.path.join(ia_resourcepack_path, "assets")
        if not os.path.exists(assets_path):
            # 婵☆偀鍋撻柡灞诲劜濡叉悂宕ラ敂鑺ョ畳models 闁?textures
            has_models = os.path.exists(os.path.join(ia_resourcepack_path, "models"))
            has_textures = os.path.exists(os.path.join(ia_resourcepack_path, "textures"))

            if has_models or has_textures:
                print(f"婵☆偀鍋撴繛鏉戭儏閸╁矂妫冮悙瀵稿灱闁告垵妫滅粊顐⑩攦閹邦剙鐦剁紓浣规尰閻庮垶鏁嶇仦缁㈠妧闁革负鍔戦崳鍝ョ磼閸曨亣绀?assets/{namespace}/...")
                # 闁告帗绋戠紓鎾寸▔閳ь剚绋夐鍛厐闁汇劌瀚径宥夊籍閸撲焦绐楃憸鐗堟磻缂嶆梹绋夋ウ璺ㄣ偒婵犙勫姇鐎垫﹢寮介崷顓熺獥鐟滅増娲╃槐婵囩閵夆晙缂夐柛蹇撶У閽栧嫰寮婚幘鍐叉枾濠殿喖顑嗚ぐ渚€宕ｉ弽顐ｇ獥鐟滅増娲橀崹銊﹀緞閸曨厽鍊為悹渚灠缁剁偤宕橀懠顒傚磹
                restructured_root = os.path.join(session_upload_dir, "restructured_rp")
                target_ns_dir = os.path.join(restructured_root, "assets", namespace)
                os.makedirs(target_ns_dir, exist_ok=True)

                # 缂佸顕ф慨鈺呭棘閸ワ附顐藉?
                for folder_name in ["models", "textures", "sounds"]:
                    src_folder = os.path.join(ia_resourcepack_path, folder_name)
                    if os.path.exists(src_folder):
                        dst_folder = os.path.join(target_ns_dir, folder_name)
                        # 缂佸顕ф慨鈺呭棘閸ワ附顐藉?
                        shutil.move(src_folder, dst_folder)

                # 闁哄洤鐡ㄩ弻濠勬導閸曨剛鐖遍柛鏍ф嚀閻儳顕ラ崟顒€鐦归柛姘灦閺屽﹪鎯冮崟顒傚灱闁告垵妫涚划銊╁几閸曨剛澹岄柣鈺婂枛缂?
                ia_resourcepack_path = restructured_root
        else:
            # 闁哄秴娲ら崳顖滅磼閹惧鈧垶鏁嶅顒夋搐闁哄绮岄幊锟犲触瀹ュ洠鏁勯梻鍌氱摠閺佸ジ宕ｅ鍫㈢閻忓繑绻嗛惁顖炴煂瀹ュ懏鍤掗柛姘У閺嬪啯绂掔捄鎭掍粴濞寸姰鍎辩亸顕€鏌婂鍡樼厐闁汇劌瀚幊锟犲触瀹ュ洠鏁勯梻?
            if namespace != original_namespace:
                src_ns_path = os.path.join(assets_path, original_namespace)
                dst_ns_path = os.path.join(assets_path, namespace)
                if os.path.exists(src_ns_path) and not os.path.exists(dst_ns_path):
                    try:
                        print(f"Renaming resource pack namespace: {original_namespace} -> {namespace}")
                        shutil.move(src_ns_path, dst_ns_path)
                    except Exception as e:
                        print(f"Warning: Failed to rename namespace folder: {e}")

    ce_output_base = os.path.join(session_output_dir, "CraftEngine", "resources", namespace)
    ce_config_dir = os.path.join(ce_output_base, "configuration", "items", namespace)
    ce_res_dir = os.path.join(ce_output_base, "resourcepack")

    # 濠碘€冲€归悘澶愬箥閹冪厒 resourcepack 闁告帗鐟ㄩ鏇犵磾椤旇法銈繝褎鍔橀惌鎯ь嚗?
    if ia_resourcepack_path:
        converter.set_resource_paths(ia_resourcepack_path, ce_res_dir)

    converter.convert(ia_data, namespace=namespace)

    converter.save_config(ce_config_dir)

    return _package_and_respond(session_output_dir, session_upload_dir, target_format)

def _package_and_respond(session_output_dir, session_upload_dir, target_format):
    # 5. 闁告ê顑囩紓澶岀磼閹惧浜?
    # 闁兼儳鍢茶ぐ鍥储閻斿娼楅柡鍌氭矗濞嗐垽宕?
    original_filename = "converted"
    try:
        for f in os.listdir(session_upload_dir):
            if f.endswith(".zip"):
                original_filename = f[:-4] # 缂佸顭峰▍?.zip
                break
    except:
        pass

    output_filename = f"{original_filename} [{target_format} by MCC].zip"
    # 缂佺姭鍋撻柛妤佹礈濞堟垿寮崶锔筋偨闁告艾绉电粩濠氭偠閸☆厾绀夐梻鍐ㄥ级椤掓盯妫冮悙瀵搞€婇悗娑欘殘椤?
    output_filename = re.sub(r'[\\/*?:"<>|]', "", output_filename)

    output_zip_path = os.path.join(app.config['OUTPUT_FOLDER'], output_filename)
    # 闁瑰瓨鍨冲鎴犳暜鐏炵偓绠块柛妯侯儑缂傚宕犻崨闈舵帡宕㈢€ｎ亝鍊甸柣鈺佺摠鐢挳寮?resources 闁哄倸娲ｅ▎銏″緞閻у摜绀夐柟瀛樼墳閳?CraftEngine 闁哄倸娲ｅ▎銏″緞?

    shutil.make_archive(output_zip_path[:-4], 'zip', session_output_dir, "CraftEngine")

    # 婵炴挸鎳愰幃濠冨濮樺磭妯堥柡鍌氭矗濞?
    # shutil.rmtree(session_upload_dir)
    # shutil.rmtree(session_output_dir)

    return jsonify({
        'status': 'success',
        'download_url': f'/api/download/{output_filename}'
    })

@app.route('/api/download/<filename>')
def download_file(filename):
    return send_file(os.path.join(app.config['OUTPUT_FOLDER'], filename), as_attachment=True)

@app.route('/api/shutdown', methods=['POST'])
def shutdown():
    """Shut down the server."""
    func = request.environ.get('werkzeug.server.shutdown')
    if func is not None:
        func()
        return jsonify({'status': 'server shutting down...'})

    def kill():
        os._exit(0)

    Timer(1.0, kill).start()
    return jsonify({'status': 'server shutting down...'})

def open_browser():
    webbrowser.open_new('http://127.0.0.1:5000/')

# 闊洤鍟抽悜锕傚礂閵娿儳婀伴柣妯垮煐閳?
last_heartbeat = time.time()
HEARTBEAT_TIMEOUT = 15  # 缂佸甯槐婵囨櫠閻愭彃顫ｉ悺鎺戞噺濡炲倿寮崼鏇燂紵濞寸姰鍎遍崢鎴犳媼閸涘﹥绾梻鈧捄銊︾暠闁告凹鍨版慨鈺呭礉閻樼儤绁?

@app.route('/api/heartbeat', methods=['POST'])
def heartbeat():
    global last_heartbeat
    last_heartbeat = time.time()
    return jsonify({'status': 'alive'})

def check_heartbeat():
    """Monitor heartbeats and stop on timeout."""
    global last_heartbeat
    while True:
        time.sleep(1)
        # 濠碘€冲€归悘?TIMEOUT 缂佸甯掗崬鏉戔柦閳╁啯绠掗煫鍥у暢閻戯箓鏁嶇仦钘夌仧闁稿繑濞婂Λ?
        if time.time() - last_heartbeat > HEARTBEAT_TIMEOUT:
            print("闊洤鍟抽悜锔炬惥閸涱喗顦ч柕鍡楀€归婊堝捶閵娿儱褰犻梻鍌ゅ幗濠€鍥礉閳ヨ櫕鐝?..")
            # 濞达綀娉曢弫?os._exit 濞寸姴娴烽崵搴ｇ矙鐎ｎ剛褰岄柛妤€纾划鎾愁潰?
            os._exit(0)

if __name__ == '__main__':
    # 濞寸姴鎳庡﹢顏堟閻愬墎娈堕悹鍥ㄦ礃鑶╃€殿喖绻嬬粭鍛村箥閹惧磭纾绘繛鏉戠箺椤秹宕?(闂佹彃绉峰ù鍥ㄥ濮橆剦鍤ら柤閿嬫綑瀵鏌屽鍡椻叺鐎殿喒鍋?
    # 濞达絽妫楅顔界鎼淬垹鈪甸柛鏍ф噽濞堟垶鎯旈弮鍌涙殢闁挎稑鐭侀惃鐔烘嫚閺団懇鍋撳顒傚煑濞?False 闁瑰瓨鐗旂粭澶愭儎缁嬪灝褰犻柕?
    if not os.environ.get("WERKZEUG_RUN_MAIN"):
        Timer(1.5, open_browser).start()

        # 闂佹彃绉堕悿鍡氱疀閸愵厾鍎查悹浣插墲濡炲倿宕抽妸銈勭鞍闂侇剙鐏濋崢銈夊捶閵娿儲鍎欓柛鏂诲妽濠€锟犳⒒绾惧孝闁?
        last_heartbeat = time.time()

        # 闁告凹鍨版慨鈺勭疀閸愵厾鍎查柣鈺傚灦鐢墎鐥捄銊㈡煠
        import threading
        monitor_thread = threading.Thread(target=check_heartbeat, daemon=True)
        monitor_thread.start()

    app.run(debug=False, port=5000)
