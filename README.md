本系统根据 兰玉磊先生的 ferry系统 进行了二次开发，对相关前端页面进行了美化，增加了部分功能，分享至此
<img width="2531" height="1259" alt="2986aa50-d5c2-429a-ba45-9bebbb491b9b" src="https://github.com/user-attachments/assets/de77ad58-ea13-48a1-8db9-291a7d154f09" />
<img width="2533" height="1261" alt="1b8f200d-a4cd-46c8-aed5-f430876b5c8c" src="https://github.com/user-attachments/assets/02583559-0d16-45d9-a4bc-61dfb592fac5" />
<img width="806" height="671" alt="bcd4b911-3572-43f6-a009-774af748bf97" src="https://github.com/user-attachments/assets/3a180cb0-2742-443b-babe-18435dc97c3f" />

##使用说明##


1.需提前准备mysql数据库及redis，在mysql中创建ferry的库
2.下载压缩包后解压，编辑 ferry/config/settings.yml 文件，将其中的mysql及redis换成自己的地址
<img width="513" height="1225" alt="image" src="https://github.com/user-attachments/assets/9fde29ef-6813-4aa0-8071-0bb0f9e12542" />



建议使用docker部署，启动命令
docker run -itd --name ferry -v /data/ferry/config:/opt/workflow/ferry/config -p 8002:8002 ferry_pro:06-24
docker hub镜像
docker push laibaxiaolayu/ferry-pro:tagname
