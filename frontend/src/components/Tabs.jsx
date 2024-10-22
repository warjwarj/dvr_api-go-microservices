import styles from './Tabs.css';

function TabButtons({ tabData, activeTab, setActiveTab }) {
  return (
    <div className="tab__header">
      {tabData.map((tab, index) => (
          <li
          className={`${index === activeTab && "active"} tab__button`}
              key={tab.title}
              onClick={() => setActiveTab(index)}
          >
          {tab.title}
        </li>
      ))}
    </div>
  );
}

function TabContent({ tabData, activeTab }) {
  return (
      <div className={styles.tab_container}>
          <div className={styles.tab_content}>
              {tabData[activeTab].component}
          </div>
      </div>
  );
}

export {
    TabButtons,
    TabContent
}