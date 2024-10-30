import React, { Component } from 'react';
import { Route, Routes, Navigate } from 'react-router-dom';
import { Layout } from './components/Layout';
import './custom.css';

import Devices from './components/Devices'

export default class App extends Component {
    static displayName = App.name;
    render() {
        return (
            <Layout>
                <Routes>
                    <Route path="/devices" element={<Devices />} />
                    <Route path="/" element={<Navigate to={"/devices"}/>} />
                </Routes>
            </Layout>
        );
    }
}