import logging
import time
from selenium import webdriver
from selenium.webdriver.edge.options import Options
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
from selenium.webdriver.common.keys import Keys
from selenium.webdriver.common.keys import Keys

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
            driver.set_page_load_timeout(30)  # 页面加载超时
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
        screenshot_path = f"resource_service_error_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.error(f"❌ 已保存错误截图：{screenshot_path}")
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

        # 2. 选择20260323_测试环境_155
        logger.info("开始选择20260323_测试环境_155")
        # 先找到20260323_测试环境_155元素
        test_env_xpath = "//span[contains(text(), '20260323_测试环境_155')]"
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
        logger.info("✅ 点击20260323_测试环境_155展开箭头成功")

        # 等待测试环境展开
        time.sleep(1)
        # 截图确认展开状态
        screenshot_path = f"environment_expanded_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存测试环境展开状态截图：{screenshot_path}")

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
        screenshot_path = f"company_expanded_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存北京电厂有限公司展开状态截图：{screenshot_path}")

        # 4. 选择虚拟机A
        logger.info("开始选择K8S_A")
        vm_xpath = "//span[contains(text(), 'K8S_A')]"
        vm_select = wait.until(
            EC.element_to_be_clickable((By.XPATH, vm_xpath))
        )
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", vm_select)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", vm_select)
        logger.info("✅ 选择K8S_A成功")

        # 等待页面响应
        time.sleep(1)

        return True

    except Exception as e:
        logger.error(f"❌ 选择环境和服务失败：{str(e)}")
        # 截图保存以便调试
        screenshot_path = f"select_error_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.error(f"❌ 已保存错误截图：{screenshot_path}")
        return False


def select_task_type(driver, task_type):
    """选择任务类型"""
    try:
        wait = WebDriverWait(driver, 20)  # 增加等待时间
        
        # 1. 定位任务类型下拉框（使用用户提供的准确XPath）
        logger.info("定位任务类型下拉框")
        task_type_select_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[1]/div[1]/div[1]/div/div/div/div/div[1]/div[2]/span"
        task_type_select = wait.until(
            EC.presence_of_element_located((By.XPATH, task_type_select_xpath))
        )
        logger.info("✅ 找到任务类型下拉框")
        
        # 2. 点击展开下拉框
        logger.info("点击任务类型下拉框")
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", task_type_select)
        time.sleep(1)  # 增加等待时间
        
        # 先点击页面其他位置，确保没有其他下拉框干扰
        driver.execute_script("document.body.click();")
        time.sleep(0.5)
        
        # 截图当前状态
        screenshot_path = f"before_click_dropdown_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存点击下拉框前截图：{screenshot_path}")
        
        # 点击下拉框
        driver.execute_script("arguments[0].click();", task_type_select)
        logger.info("点击任务类型下拉框")
        
        # 等待下拉框出现
        time.sleep(1)
        
        # 截图下拉框状态
        screenshot_path = f"dropdown_opened_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存下拉框展开状态截图：{screenshot_path}")
        
        # 3. 等待下拉框展开
        logger.info("等待任务类型下拉菜单出现")
        wait.until(EC.presence_of_element_located((By.XPATH, "//div[@class='el-select-dropdown']")))
        
        # 4. 检查是否存在iframe
        logger.info("检查是否存在iframe")
        iframes = driver.find_elements(By.TAG_NAME, "iframe")
        logger.info(f"找到 {len(iframes)} 个iframe")
        
        # 5. 查找所有下拉选项并打印（尝试多种XPath）
        logger.info("查找所有下拉选项")
        option_xpaths = [
            "//ul[@class='el-select-dropdown__list']//li",
            "//div[@class='el-select-dropdown']//li",
            "//li",
            "//*[contains(@class, 'el-select-dropdown__list')]//*[contains(@class, 'el-select-dropdown__item')]",
            "//*[text()='定时任务']/ancestor::li"
        ]
        
        found_options = []
        for xpath in option_xpaths:
            options = driver.find_elements(By.XPATH, xpath)
            logger.info(f"使用XPath {xpath} 找到 {len(options)} 个元素")
            for i, option in enumerate(options):
                try:
                    text = option.text
                    found_options.append(text)
                    logger.info(f"选项 {i+1}: {text}")
                except Exception as e:
                    logger.warning(f"获取选项 {i+1} 文本失败：{str(e)}")
        
        if found_options:
            logger.info(f"总共找到 {len(found_options)} 个选项：{found_options}")
        else:
            logger.warning("未找到任何下拉选项")
        
        # 6. 定位并点击目标选项
        logger.info(f"定位{task_type}选项")
        # 使用多种XPath尝试定位
        task_type_option_xpaths = [
            "//*[text()='定时任务']",
            "//*[contains(text(), '定时任务')]",
            "//li[text()='定时任务']",
            "//li[contains(text(), '定时任务')]",
            "//ul[@class='el-select-dropdown__list']//li[text()='定时任务']",
            "//ul[@class='el-select-dropdown__list']//li[contains(text(), '定时任务')]",
            "//div[@class='el-select-dropdown']//li[text()='定时任务']",
            "//div[@class='el-select-dropdown']//li[contains(text(), '定时任务')]",
            "//*[contains(@class, 'el-select-dropdown__item') and text()='定时任务']",
            "//*[contains(@class, 'el-select-dropdown__item') and contains(text(), '定时任务')]"
        ]
        
        task_type_option = None
        for xpath in task_type_option_xpaths:
            try:
                task_type_option = wait.until(
                    EC.element_to_be_clickable((By.XPATH, xpath))
                )
                logger.info(f"✅ 使用XPath {xpath} 找到{task_type}选项")
                break
            except Exception as e:
                logger.warning(f"使用XPath {xpath} 定位{task_type}选项失败：{str(e)}")
        
        if not task_type_option:
            raise Exception(f"未找到{task_type}选项")
        
        # 6. 尝试多种方式选择选项
        logger.info("尝试多种方式选择选项")
        
        # 截图选项状态
        screenshot_path = f"before_selection_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存选择前截图：{screenshot_path}")
        
        click_success = False
        
        # 方式1：使用键盘操作
        try:
            logger.info("方式1：使用键盘操作")
            # 先点击下拉框，确保焦点在下拉框上
            task_type_select.click()
            time.sleep(1)
            
            # 按向下箭头键（从"应用部署"到"定时任务"需要按1次）
            task_type_select.send_keys(Keys.ARROW_DOWN)
            logger.info("按向下箭头键，选择'定时任务'")
            time.sleep(0.5)
            
            # 按回车键确认选择
            task_type_select.send_keys(Keys.ENTER)
            logger.info("按回车键确认选择")
            time.sleep(1)
            
            click_success = True
            logger.info("✅ 成功使用键盘操作选择选项")
        except Exception as e:
            logger.warning(f"键盘操作失败：{str(e)}")
        
        # 方式2：使用准确的XPath定位"定时任务"选项
        if not click_success:
            try:
                logger.info("方式2：使用准确的XPath定位定时任务选项")
                # 再次点击下拉框展开
                task_type_select.click()
                time.sleep(1)
                
                # 使用更准确的XPath定位"定时任务"选项
                定时任务_xpath = "//li[contains(@class, 'el-select-dropdown__item') and contains(span, '定时任务')]"
                定时任务选项 = wait.until(
                    EC.element_to_be_clickable((By.XPATH, 定时任务_xpath))
                )
                logger.info(f"找到定时任务选项：{定时任务选项.text}")
                
                # 点击该选项
                定时任务选项.click()
                time.sleep(1)
                click_success = True
                logger.info("✅ 成功通过准确XPath选择定时任务")
            except Exception as e:
                logger.warning(f"通过准确XPath定位失败：{str(e)}")
        
        # 方式3：使用索引XPath定位第二个选项
        if not click_success:
            try:
                logger.info("方式3：使用索引XPath定位第二个选项")
                # 再次点击下拉框展开
                task_type_select.click()
                time.sleep(1)
                
                # 使用索引XPath定位第二个选项
                第二个选项_xpath = "//ul[@class='el-select-dropdown__list']//li[2]"
                第二个选项 = wait.until(
                    EC.element_to_be_clickable((By.XPATH, 第二个选项_xpath))
                )
                logger.info(f"找到第2个选项：{第二个选项.text}")
                
                # 点击该选项
                第二个选项.click()
                time.sleep(1)
                click_success = True
                logger.info("✅ 成功通过索引XPath选择定时任务")
            except Exception as e:
                logger.warning(f"通过索引XPath定位失败：{str(e)}")
        
        # 方式4：使用JavaScript直接操作
        if not click_success:
            try:
                logger.info("方式4：使用JavaScript直接操作")
                # 再次点击下拉框展开
                task_type_select.click()
                time.sleep(1)
                
                # 使用JavaScript查找并点击包含"定时任务"文本的元素
                script = """
                    var elements = document.querySelectorAll('li.el-select-dropdown__item');
                    for (var i = 0; i < elements.length; i++) {
                        if (elements[i].textContent && elements[i].textContent.includes('定时任务')) {
                            elements[i].click();
                            return true;
                        }
                    }
                    return false;
                """
                result = driver.execute_script(script)
                if result:
                    logger.info("✅ 成功使用JavaScript选择定时任务")
                    click_success = True
                    time.sleep(1)
                else:
                    logger.warning("JavaScript操作失败，未找到定时任务选项")
            except Exception as e:
                logger.warning(f"JavaScript操作失败：{str(e)}")
        
        # 方式5：通过索引定位并点击
        if not click_success:
            try:
                logger.info("方式5：通过索引定位并点击")
                # 再次点击下拉框展开
                task_type_select.click()
                time.sleep(1)
                
                # 定位所有下拉选项
                options = driver.find_elements(By.XPATH, "//ul[@class='el-select-dropdown__list']//li")
                logger.info(f"找到 {len(options)} 个下拉选项")
                
                if len(options) >= 2:  # 确保有至少2个选项（应用部署、定时任务）
                    # 选择第二个选项（索引1，因为从0开始）
                    定时任务选项 = options[1]
                    logger.info(f"选择第2个选项：{定时任务选项.text}")
                    
                    # 点击该选项
                    定时任务选项.click()
                    time.sleep(1)
                    click_success = True
                    logger.info("✅ 成功通过索引选择定时任务")
                else:
                    logger.warning("下拉选项数量不足")
            except Exception as e:
                logger.warning(f"通过索引定位失败：{str(e)}")
        
        # 截图选择后状态
        screenshot_path = f"after_selection_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存选择后截图：{screenshot_path}")
        
        # 7. 验证选择是否成功
        time.sleep(1)
        
        # 截图选择后状态
        screenshot_path = f"after_click_option_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存点击{task_type}选项后的截图：{screenshot_path}")
        
        # 检查下拉框是否已关闭
        if len(driver.find_elements(By.XPATH, "//div[@class='el-select-dropdown']")) == 0:
            logger.info(f"✅ 成功选择任务类型：{task_type}，下拉框已关闭")
        else:
            logger.warning(f"⚠️ 选择{task_type}后下拉框未关闭，可能选择失败")
            # 尝试再次点击页面其他位置
            driver.execute_script("document.body.click();")
            time.sleep(0.5)
            if len(driver.find_elements(By.XPATH, "//div[@class='el-select-dropdown']")) == 0:
                logger.info("✅ 点击页面其他位置后下拉框已关闭")
        
        return click_success
        
    except Exception as e:
        logger.error(f"❌ 选择任务类型失败：{str(e)}")
        # 截图保存以便调试
        screenshot_path = f"task_type_error_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.error(f"❌ 已保存错误截图：{screenshot_path}")
        # 打印更多调试信息
        logger.error(f"错误类型：{type(e).__name__}")
        import traceback
        logger.error(f"堆栈信息：{traceback.format_exc()}")
        return False


def click_new_task_button(driver):
    """点击新建任务按钮并填写表单"""
    try:
        wait = WebDriverWait(driver, WAIT_TIME)

        # 点击新建任务按钮
        logger.info("开始点击新建任务按钮")
        new_task_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[1]/div/div[1]/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[1]/td[7]/div/button[3]/span"
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
        
        # 选择任务类型为定时任务
        logger.info("开始选择任务类型为定时任务")
        # 截图当前状态，便于调试
        screenshot_path = f"before_task_type_click_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存任务类型操作前截图：{screenshot_path}")
        
        # 调用select_task_type函数选择定时任务
        task_type = "定时任务"
        if not select_task_type(driver, task_type):
            raise Exception("选择任务类型失败")
        
        # 截图当前状态，确认选择成功
        screenshot_path = f"task_type_selected_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存任务类型选择成功截图：{screenshot_path}")

        # 填写执行Agent
        logger.info("开始选择执行Agent")
        time.sleep(3)  # 等待页面渲染，确保元素可交互
        # 截图当前状态，便于调试
        screenshot_path = f"before_agent_click_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存执行Agent操作前截图：{screenshot_path}")

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
        screenshot_path = f"agent_dropdown_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存执行Agent下拉菜单截图：{screenshot_path}")

        # 6. 定位并选择执行Agent选项
        logger.info("定位执行Agent选项")

        # 使用最有效的XPath路径定位执行Agent选项
        agent_option_xpath = "//*[contains(text(), '10.220.42.155')]"
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
        
        # 填写Cron表达式（10秒执行一次）
        logger.info("开始填写Cron表达式")
        # 使用更通用的XPath路径定位Cron表达式输入框
        cron_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[2]/div[1]/div[1]/div/div/div[1]/div/input"
        cron_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, cron_xpath))
        )
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", cron_input)
        
        # 方法1：使用send_keys，逐个字符输入，确保空格被正确处理
        logger.info("使用send_keys方法填写Cron表达式")
        cron_input.clear()
        cron_expression = "*/1 * * * *"
        # 逐个字符输入，确保空格被正确处理
        for char in cron_expression:
            cron_input.send_keys(char)
            time.sleep(0.1)  # 每个字符之间短暂等待
        
        # 触发输入框的多个事件，确保值被正确处理
        driver.execute_script("arguments[0].dispatchEvent(new Event('input'));", cron_input)
        driver.execute_script("arguments[0].dispatchEvent(new Event('change'));", cron_input)
        driver.execute_script("arguments[0].dispatchEvent(new Event('blur'));", cron_input)
        logger.info(f"✅ 填写Cron表达式成功：{cron_expression}")
        
        # 截图当前状态，确认Cron表达式填写成功
        screenshot_path = f"cron_filled_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存Cron表达式填写成功截图：{screenshot_path}")
        
        # 验证Cron表达式是否正确填写
        time.sleep(1)
        actual_cron = cron_input.get_attribute("value")
        logger.info(f"实际填写的Cron表达式：'{actual_cron}'")
        if actual_cron == cron_expression:
            logger.info("✅ Cron表达式验证成功")
        else:
            logger.warning(f"⚠️ Cron表达式验证失败，实际值与预期不符")
            # 尝试方法2：使用JavaScript设置值
            logger.info("尝试使用JavaScript设置Cron表达式值")
            driver.execute_script(f"arguments[0].value = '{cron_expression}';", cron_input)
            driver.execute_script("arguments[0].dispatchEvent(new Event('input'));", cron_input)
            driver.execute_script("arguments[0].dispatchEvent(new Event('change'));", cron_input)
            # 再次验证
            time.sleep(1)
            actual_cron = cron_input.get_attribute("value")
            logger.info(f"再次验证Cron表达式：'{actual_cron}'")
            if actual_cron == cron_expression:
                logger.info("✅ 方法2：Cron表达式验证成功")
            else:
                logger.error(f"❌ 方法2：Cron表达式验证失败")
        
        # 直接输入开始执行时间（使用当前时间）
        logger.info("开始设置开始执行时间")
        # 定位开始执行时间输入框
        start_time_xpath = "//input[@placeholder='请选择时间']"
        start_time_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, start_time_xpath))
        )
        # 确保元素可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", start_time_input)
        # 获取当前时间并格式化为正确的格式
        start_time = time.strftime("%Y-%m-%d %H:%M:%S")
        
        # 方法1：使用send_keys输入时间
        logger.info("使用send_keys方法设置开始执行时间")
        start_time_input.clear()
        start_time_input.send_keys(start_time)
        
        # 触发多个事件，确保值被正确处理
        driver.execute_script("arguments[0].dispatchEvent(new Event('input'));", start_time_input)
        driver.execute_script("arguments[0].dispatchEvent(new Event('change'));", start_time_input)
        driver.execute_script("arguments[0].dispatchEvent(new Event('blur'));", start_time_input)
        
        # 验证开始时间是否正确设置
        time.sleep(1)
        actual_start_time = start_time_input.get_attribute("value")
        logger.info(f"实际设置的开始执行时间：'{actual_start_time}'")
        
        if actual_start_time == start_time:
            logger.info("✅ 开始执行时间设置成功")
        else:
            logger.warning("⚠️ 开始执行时间设置失败，尝试使用JavaScript设置")
            # 方法2：使用JavaScript设置值
            driver.execute_script(f"arguments[0].value = '{start_time}';", start_time_input)
            driver.execute_script("arguments[0].dispatchEvent(new Event('input'));", start_time_input)
            driver.execute_script("arguments[0].dispatchEvent(new Event('change'));", start_time_input)
            
            # 再次验证
            time.sleep(1)
            actual_start_time = start_time_input.get_attribute("value")
            logger.info(f"再次验证开始执行时间：'{actual_start_time}'")
            if actual_start_time == start_time:
                logger.info("✅ 方法2：开始执行时间设置成功")
            else:
                logger.error("❌ 开始执行时间设置失败")
        
        # 截图当前状态，确认时间设置成功
        screenshot_path = f"time_set_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存开始执行时间设置成功截图：{screenshot_path}")

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
        screenshot_path = f"plugin_window_{int(time.time())}.png"
        driver.save_screenshot(screenshot_path)
        logger.info(f"已保存插件选择窗口截图：{screenshot_path}")

        # 选择k8scapture插件
        logger.info("开始选择k8scapture插件")
        # 尝试多种方式定位k8scapture插件按钮

        k8s_plugin_xpaths = [
            "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[10]/div/div/div/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[7]/td[9]/div/button/span"
            # "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[10]/div/div/div/div[2]/div[1]/div[3]/div/div[1]/div/table/tbody/tr[1]/td[9]/div/button/span",  # 原始路径
            # "//button[contains(text(), '选择')]",  # 通过文本定位
            # "//table/tbody/tr[1]/td[last()]/div/button"  # 通过表格结构定位
        ]

        k8scapture_plugin_button = None
        for xpath in k8s_plugin_xpaths:
            try:
                logger.info(f"尝试使用XPath {xpath} 定位k8scapture插件按钮")
                k8scapture_plugin_button = wait.until(
                    EC.element_to_be_clickable((By.XPATH, xpath))
                )
                logger.info("✅ 找到k8scapture插件按钮")
                break
            except Exception as e:
                logger.warning(f"使用XPath {xpath} 定位dockercapture插件按钮失败：{str(e)}")

        if not k8scapture_plugin_button:
            raise Exception("没有找到k8scapture插件按钮")

        # 确保按钮可见
        driver.execute_script("arguments[0].scrollIntoView({block: 'center'});", k8scapture_plugin_button)
        # 使用JavaScript点击，避免可能的点击拦截
        driver.execute_script("arguments[0].click();", k8scapture_plugin_button)
        logger.info("✅ 选择k8scapture插件成功")

        # 等待插件选择窗口关闭
        logger.info("等待插件选择窗口关闭")
        try:
            wait.until(EC.invisibility_of_element_located((By.XPATH, "//*[contains(text(), '选择任务插件')]")))
        except TimeoutException:
            # 尝试使用通用的对话框定位器
            wait.until(EC.invisibility_of_element_located((By.XPATH, "//div[contains(@class, 'el-dialog')]")))
        logger.info("✅ 插件选择窗口关闭成功")

        # 填写插件标识（k8scapture）
        logger.info("开始填写插件标识")
        plugin_id_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[2]/div[6]/div/div/div/input"
        plugin_id_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, plugin_id_xpath))
        )
        plugin_id_input.clear()
        plugin_id_input.send_keys("k8scapture")
        logger.info("✅ 填写插件标识成功")

        # 填写Command（echo 'hello world'）
        logger.info("开始填写Command")
        command_xpath = "/html/body/div[1]/div/section/section/main/div[3]/div/section/main/div[8]/div/div/div/div/form/div[4]/div[2]/div[8]/div/div[1]/textarea"
        command_input = wait.until(
            EC.element_to_be_clickable((By.XPATH, command_xpath))
        )
        command_input.clear()
        # command_input.send_keys("k8sinfo")
        command_input.send_keys("k8sns")
        # command_input.send_keys("k8stop")
        # command_input.send_keys("k8scm")
        # command_input.send_keys("k8sstatefulset")
        # command_input.send_keys("k8sdeployment")
        # command_input.send_keys("k8sdaemonset")
        # command_input.send_keys("k8ssvc")
        # command_input.send_keys("k8spods")
        # command_input.send_keys("k8snodes")
        logger.info("✅ 填写Command成功")

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
            time.sleep(5)  # 5分钟等待时间
            logger.info("🔚 等待时间结束，关闭浏览器")
        finally:
            driver.quit()
    else:
        logger.error("🔚 自动化流程终止：登录失败")