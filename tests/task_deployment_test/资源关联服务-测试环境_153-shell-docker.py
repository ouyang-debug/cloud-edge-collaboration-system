import logging
import time
from selenium.webdriver.edge.options import Options
from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.edge.service import Service
from selenium.common.exceptions import (
    NoSuchElementException,
    ElementClickInterceptedException,
    TimeoutException,
    WebDriverException
)

# ====================== 日志配置 ======================
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('../login_automation.log', encoding='utf-8'),  # 日志文件
        logging.StreamHandler()  # 控制台输出
    ]
)
logger = logging.getLogger(__name__)

# ====================== 核心配置 ======================
DRIVER_PATH = r'D:\edgedriver\edgedriver_win64\msedgedriver.exe'  # 替换成你的驱动路径
LOGIN_URL = 'https://10.220.42.152:8080'  # 替换成实际登录页URL
USERNAME = 'admin'  # 用户名
PASSWORD = ''  # 密码
MAX_RETRY = 3  # 最大重试次数
WAIT_TIME = 15  # 元素等待时间


# ====================== 登录函数 ======================
def login_with_retry():
    driver = None
    retry_count = 0

    while retry_count < MAX_RETRY:
        try:
            # 初始化浏览器
            service = Service(executable_path=DRIVER_PATH)
            # driver = webdriver.Edge(service=service)
            options = Options()
            # 忽略证书错误
            options.add_argument("--ignore-certificate-errors")
            options.add_argument("--ignore-ssl-errors")
            driver = webdriver.Edge(options=options)
            driver.set_page_load_timeout(300)  # 页面加载超时
            driver.maximize_window()
            logger.info(f"第 {retry_count + 1} 次尝试登录，打开页面：{LOGIN_URL}")

            # 打开登录页
            driver.get(LOGIN_URL)
            wait = WebDriverWait(driver, WAIT_TIME)

            # 1. 输入用户名
            username_input = wait.until(
                EC.presence_of_element_located((By.XPATH, "//input[@placeholder='请输入用户名']"))
            )
            username_input.clear()
            username_input.send_keys(USERNAME)
            logger.info("✅ 用户名输入完成")

            # 2. 输入密码
            password_input = wait.until(
                EC.presence_of_element_located((By.XPATH, "//input[@placeholder='请输入密码']"))
            )
            password_input.send_keys(PASSWORD)
            logger.info("✅ 密码输入完成")

            # 3. 点击登录按钮（带重试的点击逻辑）
            login_btn = wait.until(
                EC.element_to_be_clickable((By.CSS_SELECTOR, "button.login-btn"))
            )
            login_btn.click()
            logger.info("✅ 登录按钮点击完成")

            # 4. 验证登录成功（两种判断方式，选其一）
            # 方式1：判断URL变化
            wait.until(EC.url_changes(LOGIN_URL))
            # 方式2：判断首页特征元素（替换成实际首页元素）
            # wait.until(EC.presence_of_element_located((By.XPATH, "//h1[text()='首页']")))

            logger.info("🎉 登录成功！当前URL：%s", driver.current_url)
            return driver  # 返回driver供后续操作

        except TimeoutException:
            retry_count += 1
            logger.error(f"❌ 元素加载超时，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)  # 重试前等待

        except (NoSuchElementException, ElementClickInterceptedException):
            retry_count += 1
            logger.error(f"❌ 元素定位/点击失败，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)

        except WebDriverException as e:
            retry_count += 1
            logger.error(f"❌ 浏览器驱动异常：{str(e)}，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)

        except Exception as e:
            retry_count += 1
            logger.error(f"❌ 未知异常：{str(e)}，剩余重试次数：{MAX_RETRY - retry_count}")
            if driver:
                driver.quit()
            time.sleep(2)

    logger.critical("❌ 登录失败：已重试 %d 次仍未成功", MAX_RETRY)
    if driver:
        driver.quit()
    return None


# ====================== 后续操作函数 ======================

def navigate_to_resource_association_service(driver):
    """导航到资源关联服务"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)

        # 1. 点击云边协同菜单
        logger.info("开始点击云边协同菜单")
        cloud_edge_menu = wait.until(
            EC.element_to_be_clickable((By.XPATH, "//span[text()='云边协同']"))
        )
        cloud_edge_menu.click()
        logger.info("✅ 云边协同菜单点击成功")

        # 2. 点击环境管理
        logger.info("开始点击环境管理")
        environment_management_xpath = "/html/body/div[1]/div/section/section/aside/ul/li[2]/ul/li[7]/div"
        environment_management = wait.until(
            EC.element_to_be_clickable((By.XPATH, environment_management_xpath))
        )
        environment_management.click()
        logger.info("✅ 环境管理点击成功")

        # 3. 点击资源关联服务
        logger.info("开始点击资源关联服务")
        resource_service_xpath = "/html/body/div[1]/div/section/section/aside/ul/li[2]/ul/li[7]/ul/li[4]/span"
        resource_service = wait.until(
            EC.element_to_be_clickable((By.XPATH, resource_service_xpath))
        )
        resource_service.click()
        logger.info("✅ 资源关联服务点击成功")

        # 4. 等待资源关联服务页面加载完成
        wait.until(EC.presence_of_element_located((By.XPATH, "//*[contains(text(), '资源关联服务')]")))
        logger.info("✅ 资源关联服务页面加载完成")

        return True

    except Exception as e:
        logger.error(f"❌ 导航到资源关联服务失败：{str(e)}")
        # 截图保存以便调试
        # screenshot_path = f"resource_service_error_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.error(f"❌ 已保存错误截图：{screenshot_path}")
        return False


def select_environment_and_service(driver):
    """选择环境和服务"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)

        # 1. 选择客户关系管理系统
        logger.info("开始选择客户关系管理系统")
        environment_xpath = "//span[contains(text(), '客户关系管理系统')]"
        environment_select = wait.until(
            EC.element_to_be_clickable((By.XPATH, environment_xpath))
        )
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", environment_select)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", environment_select)
        logger.info("✅ 选择客户关系管理系统成功")

        # 等待页面响应
        time.sleep(1)

        # 2. 选择测试环境_153
        logger.info("开始选择测试环境_153")
        # 先找到测试环境_153元素
        test_env_xpath = "//span[contains(text(), '测试环境_153')]"
        test_env = wait.until(
            EC.presence_of_element_located((By.XPATH, test_env_xpath))
        )
        # 找到其展开箭头
        expand_arrow = test_env.find_element(By.XPATH,
                                             "./ancestor::div[contains(@class, 'el-tree-node__content')]/i[contains(@class, 'el-tree-node__expand-icon')]")
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", expand_arrow)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", expand_arrow)
        logger.info("✅ 点击测试环境_153展开箭头成功")

        # 等待测试环境展开
        time.sleep(2)
        # 截图确认展开状态
        # screenshot_path = f"environment_expanded_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.info(f"已保存测试环境展开状态截图：{screenshot_path}")

        # 3. 选择北京电厂有限公司
        logger.info("开始选择北京电厂有限公司")
        # 先找到北京电厂有限公司元素
        company_xpath = "//span[contains(text(), '北京电厂有限公司')]"
        company = wait.until(
            EC.presence_of_element_located((By.XPATH, company_xpath))
        )
        # 找到其展开箭头
        expand_arrow = company.find_element(By.XPATH,
                                            "./ancestor::div[contains(@class, 'el-tree-node__content')]/i[contains(@class, 'el-tree-node__expand-icon')]")
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", expand_arrow)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", expand_arrow)
        logger.info("✅ 点击北京电厂有限公司展开箭头成功")

        # 等待页面响应
        time.sleep(1)
        # 截图确认北京电厂有限公司展开状态
        # screenshot_path = f"company_expanded_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.info(f"已保存北京电厂有限公司展开状态截图：{screenshot_path}")

        # 4. 选择虚拟机A
        logger.info("开始选择虚拟机A")
        vm_xpath = "//span[contains(text(), '虚拟机A')]"
        vm_select = wait.until(
            EC.element_to_be_clickable((By.XPATH, vm_xpath))
        )
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", vm_select)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", vm_select)
        logger.info("✅ 选择虚拟机A成功")

        # 等待页面响应
        time.sleep(1)

        return True

    except Exception as e:
        logger.error(f"❌ 选择环境和服务失败：{str(e)}")
        # 截图保存以便调试
        # screenshot_path = f"select_error_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.error(f"❌ 已保存错误截图：{screenshot_path}")
        return False


def click_new_task_button(driver, step_div_index=None):
    """点击新建任务按钮并填写表单"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)

        # 点击新建任务按钮
        logger.info("开始点击新建任务按钮")
        # new_task_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[1]/div/div[1]/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[1]/td[8]/div/button[3]"
        new_task_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[1]/div/div[1]/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[13]/td[7]/div/button[3]/span"
        new_task_button = wait.until(
            EC.element_to_be_clickable((By.XPATH, new_task_xpath))
        )
        # 确保按钮可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", new_task_button)
        new_task_button.click()
        logger.info("✅ 点击新建任务按钮成功")

        # 等待新建任务窗口弹出
        wait.until(EC.presence_of_element_located((By.XPATH, "//*[contains(text(), '新建任务')]")))
        logger.info("✅ 新建任务窗口弹出成功")

        # 填写执行Agent
        logger.info("开始选择执行Agent")

        # 截图当前状态，便于调试
        # screenshot_path = f"before_agent_click_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.info(f"已保存执行Agent操作前截图：{screenshot_path}")

        # 1. 找到执行Agent输入框
        logger.info("定位执行Agent输入框")
        agent_input_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[1]/div[3]/div/div/div/div[1]/div[1]/input"
        agent_input = wait.until(
            EC.presence_of_element_located((By.XPATH, agent_input_xpath))
        )
        logger.info("✅ 找到执行Agent输入框")

        # 2. 直接点击执行Agent输入框（根据用户要求）
        logger.info("点击执行Agent输入框")
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", agent_input)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", agent_input)
        logger.info("✅ 点击执行Agent输入框成功")

        # 4. 等待下拉菜单出现
        logger.info("等待执行Agent下拉菜单出现")
        wait.until(EC.presence_of_element_located((By.XPATH, "//ul[contains(@class, 'el-select-dropdown__list')]")))

        # 5. 截图当前状态，便于调试
        # screenshot_path = f"agent_dropdown_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.info(f"已保存执行Agent下拉菜单截图：{screenshot_path}")

        # 6. 定位并选择执行Agent选项
        logger.info("定位执行Agent选项")

        # 使用最有效的XPath路径定位执行Agent选项
        agent_option_xpath = "//*[contains(text(), '10.220.42.153')]"
        target_option = wait.until(
            EC.element_to_be_clickable((By.XPATH, agent_option_xpath))
        )
        logger.info("✅ 找到执行Agent选项")

        # 7. 点击选择执行Agent选项
        logger.info("点击选择执行Agent选项")
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", target_option)
        # 使用JavaScript点击，避免点击后回到输入框
        driver.execute_script("arguments[0].click();", target_option)
        logger.info("✅ 选择执行Agent成功")

        # 8. 检查执行Agent选择是否成功
        time.sleep(1)  # 短暂等待，确保值已更新
        # 检查输入框是否显示了选择的值
        input_value = agent_input.get_attribute("value")
        logger.info(f"执行Agent输入框当前值：{input_value}")
        if "10.220.42.153" in input_value:
            logger.info("✅ 执行Agent选择验证成功")
        else:
            logger.warning("⚠️ 执行Agent选择验证失败，输入框值与预期不符")

        # 填写超时时间（2秒）
        logger.info("开始填写超时时间")
        timeout_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[2]/div[2]/div[1]/div/div/div/div/div/input"
        timeout_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, timeout_xpath))
        )
        timeout_input.clear()
        timeout_input.send_keys("2")
        logger.info("✅ 填写超时时间成功")

        # 填写失败重试次数（2次）
        logger.info("开始填写失败重试次数")
        retry_count_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[2]/div[2]/div[2]/div/div/div/div/div/input"
        retry_count_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, retry_count_xpath))
        )
        retry_count_input.clear()
        retry_count_input.send_keys("2")
        logger.info("✅ 填写失败重试次数成功")

        # 填写重试间隔（2秒）
        logger.info("开始填写重试间隔")
        retry_interval_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[2]/div[3]/div[1]/div/div/div/div/div/input"
        retry_interval_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, retry_interval_xpath))
        )
        retry_interval_input.clear()
        retry_interval_input.send_keys("2")
        logger.info("✅ 填写重试间隔成功")

        # 增加步骤按钮XPath
        add_step_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[1]/button/span"
        # shell插件选择按钮XPath
        shell_plugin_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[10]/div/div/div/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[1]/td[9]/div/button/span"
        # docker插件选择按钮XPath
        docker_plugin_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[10]/div/div/div/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[10]/td[9]/div/button/span"

        # 命令配置列表 - 可以通过注释来启用或禁用特定命令
        # shell命令列表
        shell_commands = [
            "mkdir -p /data/nginx/conf /data/nginx/logs /data/nginx/html",
            "chown -R 101:101 /data/nginx",
            "chmod -R 755 /data/nginx",
            '''cat > /data/nginx/conf/default.conf << 'EOF'
server {
    listen       80;
    server_name  localhost;
    access_log  /var/log/nginx/access.log;
    error_log   /var/log/nginx/error.log;
    location / {
        root   /usr/share/nginx/html;
        index  index.html index.htm;
    }
}
EOF''',
            "chown 101:101 /data/nginx/conf/default.conf",        # 设置文件所有者
            "chmod 644 /data/nginx/conf/default.conf",   # 设置文件权限
            # "dockerimages",   # 查看Docker镜像列表（示例：已注释）
            # "dockernetwork",  # 查看Docker网络（示例：已注释）
        ]
        
        # docker命令
        docker_command = "docker-compose up -d"  # 查看Docker容器统计信息
        
        # 生成步骤配置列表
        steps_config = []
        
        # 添加shell命令步骤
        for i, command in enumerate(shell_commands, 1):
            steps_config.append({
                "name": str(i),
                "plugin": "shell",
                "command": command,
                "plugin_button_xpath": shell_plugin_button_xpath
            })
        
        # 添加docker命令步骤
        steps_config.append({
            "name": str(len(shell_commands) + 1),
            "plugin": "docker",
            "command": docker_command,
            "plugin_button_xpath": docker_plugin_button_xpath
        })

        # 处理每个步骤
        for i, step_config in enumerate(steps_config):
            step_name = step_config["name"]
            plugin_name = step_config["plugin"]
            command = step_config["command"]
            plugin_button_xpath = step_config["plugin_button_xpath"]

            logger.info(f"\n=== 处理步骤 {step_name} ===")

            # 计算步骤对应的div索引（从2开始，因为第一个步骤是div[2]）
            step_div_index = i + 2
            logger.info(f"步骤 {step_name} 对应的div索引：{step_div_index}")

            # 如果不是第一个步骤，点击增加步骤按钮
            if i > 0:
                logger.info("点击增加步骤按钮")
                add_step_button = wait.until(
                    EC.element_to_be_clickable((By.XPATH, add_step_button_xpath))
                )
                # 确保按钮可见
                driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", add_step_button)
                # 使用JavaScript点击，避免可能的点击拦截
                driver.execute_script("arguments[0].click();", add_step_button)
                logger.info("✅ 点击增加步骤按钮成功")
                # 等待步骤添加完成（增加等待时间）
                logger.info("等待3秒，确保步骤添加完成")
                time.sleep(3)
                # 验证步骤是否添加成功
                try:
                    # 尝试定位新添加的步骤的名称输入框
                    new_step_xpath = f"/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[{step_div_index}]/div[2]/div/div/div/input"
                    new_step_input = wait.until(
                        EC.presence_of_element_located((By.XPATH, new_step_xpath))
                    )
                    logger.info(f"✅ 步骤 {step_name} 添加成功，元素定位到：{new_step_xpath}")
                except Exception as e:
                    logger.error(f"❌ 步骤 {step_name} 添加失败：{str(e)}")
                    # 截图保存以便调试
                    # screenshot_path = f"step_add_error_{int(time.time())}.png"
                    # driver.save_screenshot(screenshot_path)
                    # logger.error(f"❌ 已保存错误截图：{screenshot_path}")
                    continue

            # 填写步骤名称
            logger.info("开始填写步骤名称")
            step_name_xpath = f"/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[{step_div_index}]/div[2]/div/div/div/input"
            logger.info(f"步骤名称输入框XPath：{step_name_xpath}")
            step_name_input = wait.until(
                EC.element_to_be_clickable((By.XPATH, step_name_xpath))
            )
            # 清除并填写步骤名称
            step_name_input.clear()
            step_name_input.send_keys(step_name)
            # 验证步骤名称是否填写成功
            filled_name = step_name_input.get_attribute("value")
            logger.info(f"步骤名称填写结果：{filled_name}")
            if filled_name == step_name:
                logger.info("✅ 填写步骤名称成功")
            else:
                logger.warning(f"⚠️ 步骤名称填写可能失败，实际填写：{filled_name}，预期：{step_name}")

            # 点击选择任务插件按钮
            logger.info("开始选择任务插件")
            task_plugin_button_xpath = f"/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[{step_div_index}]/div[3]/div/div/button/span"
            logger.info(f"插件选择按钮XPath：{task_plugin_button_xpath}")
            task_plugin_button = wait.until(
                EC.element_to_be_clickable((By.XPATH, task_plugin_button_xpath))
            )
            task_plugin_button.click()
            logger.info("✅ 点击选择任务插件按钮成功")

            # 等待插件选择窗口弹出
            logger.info("等待插件选择窗口弹出")
            # 使用更短的等待时间，避免长时间超时
            short_wait = WebDriverWait(driver, 2)
            try:
                # 尝试使用标题定位
                short_wait.until(EC.presence_of_element_located((By.XPATH, "//*[contains(text(), '选择任务插件')]")))
                logger.info("✅ 插件选择窗口弹出成功（使用标题定位）")
            except TimeoutException:
                # 尝试使用通用的对话框定位器
                logger.warning("⚠️ 插件选择窗口标题定位失败，尝试使用通用定位器")
                short_wait.until(EC.presence_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
                logger.info("✅ 插件选择窗口弹出成功（使用通用定位器）")

            # 选择插件
            logger.info(f"开始选择{plugin_name}插件")
            plugin_button = wait.until(
                EC.element_to_be_clickable((By.XPATH, plugin_button_xpath))
            )
            # 确保按钮可见
            driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", plugin_button)
            # 使用JavaScript点击，避免可能的点击拦截
            driver.execute_script("arguments[0].click();", plugin_button)
            logger.info(f"✅ 选择{plugin_name}插件成功")

            # 等待插件选择窗口关闭
            logger.info("等待插件选择窗口关闭")
            try:
                wait.until(EC.invisibility_of_element_located((By.XPATH, "//*[contains(text(), '选择任务插件')]")))
            except TimeoutException:
                # 尝试使用通用的对话框定位器
                wait.until(EC.invisibility_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
            logger.info("✅ 插件选择窗口关闭成功")

            # 填写插件标识
            logger.info("开始填写插件标识")
            plugin_id_xpath = f"/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[{step_div_index}]/div[6]/div/div/div/input"
            logger.info(f"插件标识输入框XPath：{plugin_id_xpath}")
            plugin_id_input = wait.until(
                EC.element_to_be_clickable((By.XPATH, plugin_id_xpath))
            )
            plugin_id_input.clear()
            plugin_id_input.send_keys(plugin_name)
            # 验证插件标识是否填写成功
            filled_plugin_id = plugin_id_input.get_attribute("value")
            logger.info(f"插件标识填写结果：{filled_plugin_id}")
            if filled_plugin_id == plugin_name:
                logger.info("✅ 填写插件标识成功")
            else:
                logger.warning(f"⚠️ 插件标识填写可能失败，实际填写：{filled_plugin_id}，预期：{plugin_name}")

            # 填写Command
            logger.info("开始填写Command")
            command_xpath = f"/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[{step_div_index}]/div[8]/div/div/textarea"
            logger.info(f"Command输入框XPath：{command_xpath}")
            command_input = wait.until(
                EC.element_to_be_clickable((By.XPATH, command_xpath))
            )
            
            # 先清除输入框（使用键盘事件）
            command_input.click()
            # 使用键盘事件清除内容（Ctrl+A + Backspace）
            driver.execute_script("arguments[0].focus();", command_input)
            driver.execute_script("arguments[0].select();", command_input)
            driver.execute_script("document.execCommand('delete', false, null);", command_input)
            
            # 使用模拟键盘输入的方式填写命令，确保触发前端事件
            logger.info(f"正在输入命令：{command}")
            # 分批次输入命令，避免一次性输入过长导致的问题
            command_parts = [command[i:i+50] for i in range(0, len(command), 50)]
            for part in command_parts:
                command_input.send_keys(part)
                # 短暂等待，确保输入被处理
                time.sleep(0.1)
            
            # 验证Command是否填写成功
            filled_command = command_input.get_attribute("value")
            logger.info(f"Command填写结果（前50个字符）：{filled_command[:50]}...")
            if filled_command == command:
                logger.info("✅ 填写Command成功")
            else:
                logger.warning(f"⚠️ Command填写可能失败，实际填写：{filled_command}，预期：{command}")
            
            # 触发输入事件，确保前端框架感知到变化
            driver.execute_script("""
                var event = new Event('input', { bubbles: true });
                arguments[0].dispatchEvent(event);
                var changeEvent = new Event('change', { bubbles: true });
                arguments[0].dispatchEvent(changeEvent);
            """, command_input)
            logger.info("✅ 触发输入事件成功")
            
            # 对于步骤7（docker插件），需要先选择文件，再填写命令
            if step_name == "7" and plugin_name == "docker":
                logger.info("开始选择配置文件")
                # 点击配置文件选择按钮
                config_file_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[8]/div[7]/div/div[1]/button"
                logger.info(f"配置文件选择按钮XPath：{config_file_button_xpath}")
                config_file_button = wait.until(
                    EC.element_to_be_clickable((By.XPATH, config_file_button_xpath))
                )
                config_file_button.click()
                logger.info("✅ 点击配置文件选择按钮成功")
                
                # 等待配置文件选择窗口弹出
                wait.until(EC.presence_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
                logger.info("✅ 配置文件选择窗口弹出成功")
                
                # 选择文件（docker-compose.yml）
                logger.info("开始选择文件")
                file_select_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[11]/div/div/div/div/div[1]/div[3]/div/div[1]/div/table/tbody/tr/td[1]/div"
                logger.info(f"文件选择XPath：{file_select_xpath}")
                file_select = wait.until(
                    EC.element_to_be_clickable((By.XPATH, file_select_xpath))
                )
                file_select.click()
                logger.info("✅ 选择文件成功")
                
                # 点击确认按钮
                logger.info("开始点击确认按钮")
                config_file_confirm_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[11]/div/div/footer/span/button[2]/span"
                logger.info(f"确认按钮XPath：{config_file_confirm_xpath}")
                config_file_confirm = wait.until(
                    EC.element_to_be_clickable((By.XPATH, config_file_confirm_xpath))
                )
                config_file_confirm.click()
                logger.info("✅ 点击确认按钮成功")
                time.sleep(5)  # 等待点击确认后的页面响应                
                # 等待配置文件选择窗口关闭
                # wait.until(EC.invisibility_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
                # logger.info("✅ 配置文件选择窗口关闭成功")
                
                # 填写Command（在选择文件后）
                logger.info("开始填写Command")
                command_xpath = f"/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[{step_div_index}]/div[8]/div/div/textarea"
                logger.info(f"Command输入框XPath：{command_xpath}")
                command_input = wait.until(
                    EC.element_to_be_clickable((By.XPATH, command_xpath))
                )
                
                # 先清除输入框（使用键盘事件）
                command_input.click()
                # 使用键盘事件清除内容（Ctrl+A + Backspace）
                driver.execute_script("arguments[0].focus();", command_input)
                driver.execute_script("arguments[0].select();", command_input)
                driver.execute_script("document.execCommand('delete', false, null);", command_input)
                
                # 使用模拟键盘输入的方式填写命令，确保触发前端事件
                logger.info(f"正在输入命令：{command}")
                # 分批次输入命令，避免一次性输入过长导致的问题
                command_parts = [command[i:i+50] for i in range(0, len(command), 50)]
                for part in command_parts:
                    command_input.send_keys(part)
                    # 短暂等待，确保输入被处理
                    time.sleep(1)
                
                # 验证Command是否填写成功
                filled_command = command_input.get_attribute("value")
                logger.info(f"Command填写结果（前50个字符）：{filled_command[:50]}...")
                if filled_command == command:
                    logger.info("✅ 填写Command成功")
                else:
                    logger.warning(f"⚠️ Command填写可能失败，实际填写：{filled_command}，预期：{command}")
                
                # 触发输入事件，确保前端框架感知到变化
                driver.execute_script("""
                    var event = new Event('input', { bubbles: true });
                    arguments[0].dispatchEvent(event);
                    var changeEvent = new Event('change', { bubbles: true });
                    arguments[0].dispatchEvent(changeEvent);
                """, command_input)
                logger.info("✅ 触发输入事件成功")

                # 添加等待时间，确保所有步骤配置完成
                logger.info("等待2秒，确保所有步骤配置完成")
                time.sleep(2)
                
                # 点击确认按钮
                logger.info("开始点击确认按钮")
                confirm_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/footer/span/button[2]"
                confirm_button = wait.until(
                    EC.element_to_be_clickable((By.XPATH, confirm_button_xpath))
                )
                # 确保按钮可见
                driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", confirm_button)
                time.sleep(1)
                # 使用JavaScript点击，避免可能的点击拦截
                # driver.execute_script("arguments[0].click();", confirm_button)
                driver.execute_script("""
            // 触发鼠标悬停
            var event = new MouseEvent('mouseover', {bubbles: true, cancelable: true, view: window});
            arguments[0].dispatchEvent(event);
            
            // 触发鼠标按下
            event = new MouseEvent('mousedown', {bubbles: true, cancelable: true, view: window});
            arguments[0].dispatchEvent(event);
            
            // 触发点击
            event = new MouseEvent('click', {bubbles: true, cancelable: true, view: window});
            arguments[0].dispatchEvent(event);
            
            // 触发鼠标释放
            event = new MouseEvent('mouseup', {bubbles: true, cancelable: true, view: window});
            arguments[0].dispatchEvent(event);
        """, confirm_button)
                logger.info("✅ 点击确认按钮成功")
                # 增加等待时间，确保页面跳转
                logger.info("等待页面跳转...")
                time.sleep(5)
                # 等待任务创建成功
                logger.info("等待任务创建成功")
                wait.until(EC.invisibility_of_element_located((By.XPATH, "//*[contains(text(), '新建任务')]")))
                logger.info("✅ 任务创建完成")

                return True
    except Exception as e:
        logger.error(f"❌ 点击新建任务按钮失败：{str(e)}")
        # 截图保存以便调试
        screenshot_path = f"new_task_error_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.error(f"❌ 已保存错误截图：{screenshot_path}")
        return False   

# ====================== 主函数 ======================
if __name__ == "__main__":
    # 执行登录
    driver = login_with_retry()

    # 登录成功后的后续操作
    if driver:
        try:
            # 导航到资源关联服务
            if navigate_to_resource_association_service(driver):
                # 选择环境和服务
                if select_environment_and_service(driver):
                    # 点击新建任务按钮
                    click_new_task_button(driver)

            # 操作完成后不关闭浏览器，让用户手动操作
            logger.info("🔚 自动化流程完成，等待用户手动操作")
            logger.info("请维护必要信息后点击确认按钮")
            # 停留一段时间让用户操作
            time.sleep(2)  # 5分钟等待时间
            logger.info("🔚 等待时间结束，关闭浏览器")
        finally:
            driver.quit()
    else:
        logger.error("🔚 自动化流程终止：登录失败")