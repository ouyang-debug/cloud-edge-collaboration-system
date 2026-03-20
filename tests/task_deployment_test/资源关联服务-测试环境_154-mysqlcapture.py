import logging
import time
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
LOGIN_URL = 'http://10.220.42.152:8080'  # 替换成实际登录页URL
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
            driver = webdriver.Edge(service=service)
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

        # 2. 选择测试环境_154
        logger.info("开始选择测试环境_154")
        # 先找到测试环境_154元素
        test_env_xpath = "//span[contains(text(), '测试环境_154')]"
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
        logger.info("✅ 点击测试环境_154展开箭头成功")

        # 等待测试环境展开
        time.sleep(1)
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

        # 4. 选择虚拟机B
        logger.info("开始选择mysql_A")
        vm_xpath = "//span[contains(text(), 'mysql_A')]"
        vm_select = wait.until(
            EC.element_to_be_clickable((By.XPATH, vm_xpath))
        )

        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", vm_select)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", vm_select)
        logger.info("✅ 选择mysql_A成功")

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


def click_new_task_button(driver):
    """点击新建任务按钮并填写表单"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)

        # 点击新建任务按钮
        logger.info("开始点击新建任务按钮")
        new_task_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[1]/div/div[1]/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[1]/td[8]/div/button[3]"
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
        agent_option_xpath = "//*[contains(text(), '10.220.42.154')]"
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

        # 填写步骤名称（1）
        logger.info("开始填写步骤名称")
        step_name_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[2]/div[2]/div/div/div/input"
        step_name_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, step_name_xpath))
        )
        step_name_input.clear()
        step_name_input.send_keys("1")
        logger.info("✅ 填写步骤名称成功")

        # 点击选择任务插件按钮
        logger.info("开始选择任务插件")
        task_plugin_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[2]/div[3]/div/div/button/span"
        task_plugin_button = wait.until(
            EC.element_to_be_clickable((By.XPATH, task_plugin_button_xpath))
        )
        task_plugin_button.click()
        logger.info("✅ 点击选择任务插件按钮成功")

        # 等待插件选择窗口弹出
        logger.info("等待插件选择窗口弹出")
        # 使用更短的等待时间，避免长时间超时
        short_wait = WebDriverWait(driver, 2)  # 5秒等待
        try:
            # 尝试使用标题定位
            short_wait.until(EC.presence_of_element_located((By.XPATH, "//*[contains(text(), '选择任务插件')]")))
            logger.info("✅ 插件选择窗口弹出成功（使用标题定位）")
        except TimeoutException:
            # 尝试使用通用的对话框定位器
            logger.warning("⚠️ 插件选择窗口标题定位失败，尝试使用通用定位器")
            short_wait.until(EC.presence_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
            logger.info("✅ 插件选择窗口弹出成功（使用通用定位器）")

        # 截图当前状态，便于调试
        # screenshot_path = f"plugin_window_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.info(f"已保存插件选择窗口截图：{screenshot_path}")

        # 选择mysqlcapture插件
        logger.info("开始选择mysqlcapture插件")

        #mysqlcapture插件
        mysqlcapture_plugin_xpaths = [
            "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[10]/div/div/div/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[5]/td[9]/div/button/span"
        ]


        mysqlcapture_plugin_button = None
        for xpath in mysqlcapture_plugin_xpaths:
            try:
                logger.info(f"尝试使用XPath {xpath} 定位mysqlcapture插件按钮")
                mysqlcapture_plugin_button = wait.until(
                    EC.element_to_be_clickable((By.XPATH, xpath))
                )
                logger.info("✅ 找到mysqlcapture插件按钮")
                break
            except Exception as e:
                logger.warning(f"使用XPath {xpath} 定位mysqlcapture插件按钮失败：{str(e)}")

        if not mysqlcapture_plugin_button:
            raise Exception("没有找到mysqlcapture插件按钮")

        # 确保按钮可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", mysqlcapture_plugin_button)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", mysqlcapture_plugin_button)
        logger.info("✅ 选择mysqlcapture插件成功")

        # 等待插件选择窗口关闭
        logger.info("等待插件选择窗口关闭")
        try:
            wait.until(EC.invisibility_of_element_located((By.XPATH, "//*[contains(text(), '选择任务插件')]")))
        except TimeoutException:
            # 尝试使用通用的对话框定位器
            wait.until(EC.invisibility_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
        logger.info("✅ 插件选择窗口关闭成功")

        # 填写插件标识（mysqlcapture）
        logger.info("开始填写插件标识")
        plugin_id_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[2]/div[6]/div/div/div/input"
        plugin_id_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, plugin_id_xpath))
        )
        plugin_id_input.clear()
        plugin_id_input.send_keys("mysqlcapture")
        logger.info("✅ 填写插件标识成功")

        # 填写Command（echo 'hello world'）
        logger.info("开始填写Command")
        command_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[2]/div[8]/div/div[1]/textarea"
        command_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, command_xpath))
        )
        command_input.clear()
        command_input.send_keys("db_info")
        logger.info("✅ 填写Command成功")
        # time.sleep(5)  # 5秒钟等待时间

        # 点击确认按钮
        logger.info("开始点击确认按钮")
        confirm_button_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/footer/span/button[2]/span"
        confirm_button = wait.until(
            EC.element_to_be_clickable((By.XPATH, confirm_button_xpath))
        )
        # 确保按钮可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", confirm_button)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", confirm_button)
        logger.info("✅ 点击确认按钮成功")

        # 等待任务创建成功
        logger.info("等待任务创建成功")
        wait.until(EC.invisibility_of_element_located((By.XPATH, "//*[contains(text(), '新建任务')]")))
        logger.info("✅ 任务创建完成")

        return True

    except Exception as e:
        logger.error(f"❌ 点击新建任务按钮失败：{str(e)}")
        # 截图保存以便调试
        # screenshot_path = f"new_task_error_{int(time.time())}.png"
        # driver.save_screenshot(screenshot_path)
        # logger.error(f"❌ 已保存错误截图：{screenshot_path}")
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
            time.sleep(3)  # 5分钟等待时间
            logger.info("🔚 等待时间结束，关闭浏览器")
        finally:
            driver.quit()
    else:
        logger.error("🔚 自动化流程终止：登录失败")